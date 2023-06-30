package ionos

import (
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/plan"
)

func GetCreateDeleteSetsFromChanges(changes *plan.Changes) ([]*endpoint.Endpoint, []*endpoint.Endpoint) {
	toCreate := make([]*endpoint.Endpoint, len(changes.Create))
	copy(toCreate, changes.Create)

	toDelete := make([]*endpoint.Endpoint, len(changes.Delete))
	copy(toDelete, changes.Delete)

	for i, updateOldEndpoint := range changes.UpdateOld {
		updateNewEndpoint := changes.UpdateNew[i]
		if endpointsAreDifferent(*updateOldEndpoint, *updateNewEndpoint) {
			toDelete = append(toDelete, updateOldEndpoint)
			toCreate = append(toCreate, updateNewEndpoint)
		}
	}
	return toCreate, toDelete
}

func endpointsAreDifferent(a endpoint.Endpoint, b endpoint.Endpoint) bool {
	return a.DNSName != b.DNSName || a.RecordType != b.RecordType ||
		a.RecordTTL != b.RecordTTL || !a.Targets.Same(b.Targets)
}

type EndpointCollection struct {
	epForAllRecords map[string]*endpoint.Endpoint
}

func NewEndpointCollection[R any](records []R, creator func(R) *endpoint.Endpoint, identifier func(R) string) *EndpointCollection {
	epc := &EndpointCollection{
		epForAllRecords: make(map[string]*endpoint.Endpoint, 0),
	}
	for _, record := range records {
		ep := creator(record)
		key := identifier(record)
		if existingEp, ok := epc.epForAllRecords[key]; ok {
			existingEp.Targets = append(existingEp.Targets, ep.Targets...)
		} else {
			epc.epForAllRecords[key] = ep
		}
	}
	return epc
}

func (epc *EndpointCollection) RetrieveEndPoints() []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, 0)
	for _, ep := range epc.epForAllRecords {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

type RecordCollection[R any] struct {
	records map[*endpoint.Endpoint][]R
}

func NewRecordCollection[R any](endpoints []*endpoint.Endpoint, creator func(*endpoint.Endpoint) []R) *RecordCollection[R] {
	rc := &RecordCollection[R]{records: make(map[*endpoint.Endpoint][]R, 0)}
	for _, ep := range endpoints {
		records := creator(ep)
		rc.records[ep] = records
	}
	return rc
}

func (c *RecordCollection[R]) ForEach(visit func(*endpoint.Endpoint, R) error) error {
	for ep, records := range c.records {
		for _, record := range records {
			if err := visit(ep, record); err != nil {
				return err
			}
		}
	}
	return nil
}

type zoneNode[Z any] struct {
	zone     Z
	parent   *zoneNode[Z]
	children map[string]*zoneNode[Z]
}

func (d *zoneNode[Z]) visitZoneNodesByName(name string, visitor func(*zoneNode[Z])) {
	currentNode := d
	nameParts := strings.Split(name, ".")
	for i := len(nameParts) - 1; i >= 0; i-- {
		namePart := nameParts[i]
		if currentNode.children[namePart] == nil {
			return
		}
		currentNode = currentNode.children[namePart]
		visitor(currentNode)
	}
}

func (d *zoneNode[Z]) addChild(zone Z, namePart string) {
	child := &zoneNode[Z]{
		zone:     zone,
		parent:   d,
		children: make(map[string]*zoneNode[Z]),
	}
	d.children[namePart] = child
}

func (d *zoneNode[Z]) addZone(z Z, sub string) {
	parts := strings.Split(sub, ".")
	lastPart := parts[len(parts)-1]
	if d.children[lastPart] == nil {
		if len(parts) == 1 {
			d.addChild(z, lastPart)
			return
		}
		node := &zoneNode[Z]{parent: d, children: make(map[string]*zoneNode[Z])}
		d.children[lastPart] = node
	}
	d.children[lastPart].addZone(z, strings.Join(parts[:len(parts)-1], "."))
}

func (t *ZoneTree[Z]) AddZone(zone Z, domainName string) {
	t.root.addZone(zone, domainName)
	t.count++
}

// FindZoneByDomainName returns the zone that matches the given domain name.
func (t *ZoneTree[Z]) FindZoneByDomainName(domainName string) Z {
	var result Z
	t.root.visitZoneNodesByName(domainName, func(node *zoneNode[Z]) {
		result = node.zone
	})
	return result
}

func (t *ZoneTree[Z]) GetZonesCount() int {
	return t.count
}

type ZoneTree[Z any] struct {
	count int
	root  *zoneNode[Z]
}

// NewZoneTree creates a new ZoneTree.
func NewZoneTree[Z any]() *ZoneTree[Z] {
	return &ZoneTree[Z]{
		count: 0,
		root: &zoneNode[Z]{
			children: make(map[string]*zoneNode[Z]),
		},
	}
}
