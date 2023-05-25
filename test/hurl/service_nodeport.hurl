# test use case: annotated service of type NodePort
# When a service of type NodePort is annotated with "external-dns.alpha.kubernetes.io/internal-hostname",
# then DNS records are created in the zone of the annotation value with the node ip as content 


# --- PREPARE ---

# delete annotation of the servive of ingress-nginx-controller
PATCH {{k8s_api_url}}/api/v1/namespaces/ingress-nginx/services/ingress-nginx-controller
Content-Type: application/merge-patch+json

{
  "metadata": {
    "annotations": {
      "external-dns.alpha.kubernetes.io/internal-hostname": null
    }
  }
}

HTTP/1.1 200

# reset
PUT {{dns_mockserver_url}}/mockserver/reset
accept : */*

HTTP/1.1 200

# --- GIVEN ---

# expectations

# get zones
PUT {{dns_mockserver_url}}/mockserver/expectation
Content-Type : application/json

{
  "httpRequest" : {
    "method" : "GET",
    "path" : "/v1/zones"
  },
  "httpResponse" : {
    "body" : [
                {
                  "id": "{{zone_id}}",
                  "name": "test.com",
                  "type": "NATIVE"
                }
            ]
  }
}

HTTP/1.1 201
[Captures]
getZonesExpectationId : jsonpath "$[0].id"

# get zone
PUT {{dns_mockserver_url}}/mockserver/expectation
Content-Type : application/json

{
  "httpRequest" : {
    "method" : "GET",
    "path" : "/v1/zones/{{zone_id}}"
  },
  "httpResponse" : {
    "body" :
          { 
            "id": "{{zone_id}}",
            "name": "test.com",
            "type": "NATIVE",
            "records": [
              {
                "id": "{{record_id}}",
                "name": "test.com",
                "rootName": "test.com",
                "type": "A",
                "content": "1.1.1.1",
                "changeDate": "2019-12-09T13:04:25.772Z",
                "ttl": 3600,
                "prio": 0,
                "disabled": false
              }
            ]
          }
  }
}

HTTP/1.1 201
[Captures]
getZoneExpectationId : jsonpath "$[0].id"



# get node ip
GET {{k8s_api_url}}/api/v1/nodes
Content-Type: application/json

HTTP/1.1 200
[Captures]
node_ip: jsonpath "$.items[0].status.addresses[?(@.type=='InternalIP')].address"


# create A record 
PUT {{dns_mockserver_url}}/mockserver/expectation
Content-Type : application/json

{
  "httpRequest": {
    "method": "POST",
    "path": "/v1/zones/{zone_id}/records",
    "pathParameters": {
        "zone_id": ["{{zone_id}}"]
    },
    "headers" : {
      "Content-Type" : [ "application/json" ],
      "Accept" : [ "application/json" ],
      "X-Api-Key" : [ "test-api-key" ]
    },
    "body" : {
      "type": "JSON",
      "json": [ 
        {
          "content" : "{{node_ip}}",
          "disabled" : false,
          "name" : "example.test.com",
          "type" : "A"
        } 
      ],
      "matchType": "ONLY_MATCHING_FIELDS"
    }
  },
  "httpResponse" : {
    "statusCode" : 201,
    "body" : [
                {
                  "id": "1234",
                  "name": "example.test.com",
                  "rootName": "example.test.com",
                  "type": "A",
                  "content": "{{node_ip}}",
                  "changeDate": "2022-23-03T13:04:25.772Z",
                  "ttl": 3600,
                  "prio": 0,
                  "disabled": false
                }
             ]
  }
}

HTTP/1.1 201
[Captures]
createARecordExpectationId : jsonpath "$[0].id"

# create txt record 
PUT {{dns_mockserver_url}}/mockserver/expectation
Content-Type : application/json

{
  "httpRequest": {
    "method": "POST",
    "path": "/v1/zones/{zone_id}/records",
    "pathParameters": {
        "zone_id": ["{{zone_id}}"]
    },
    "headers" : {
      "Content-Type" : [ "application/json" ],
      "Accept" : [ "application/json" ],
      "X-Api-Key" : [ "test-api-key" ]
    },
    "body" : {
      "type": "JSON",
      "json": [ 
        {
          "content" : "\"heritage=external-dns,external-dns/owner=default,external-dns/resource=service/ingress-nginx/ingress-nginx-controller\"",
          "disabled" : false,
          "name" : "example.test.com",
          "type" : "TXT"
        } 
      ],
      "matchType": "ONLY_MATCHING_FIELDS"
    }
  },
  "httpResponse" : {
    "statusCode" : 201,
    "body" : [
                {
                  "id": "1234",
                  "name": "example.test.com",
                  "rootName": "example.test.com",
                  "type": "TXT",
                  "content": "\"heritage=external-dns,external-dns/owner=default,external-dns/resource=service/ingress-nginx/ingress-nginx-controller\"",
                  "changeDate": "2022-23-03T13:04:25.772Z",
                  "ttl": 3600,
                  "prio": 0,
                  "disabled": false
                }
             ]
  }
}

HTTP/1.1 201
[Captures]
createTXTRecordExpectationId : jsonpath "$[0].id"

# create a-txt record 
PUT {{dns_mockserver_url}}/mockserver/expectation
Content-Type : application/json

{
  "httpRequest": {
    "method": "POST",
    "path": "/v1/zones/{zone_id}/records",
    "pathParameters": {
        "zone_id": ["{{zone_id}}"]
    },
    "headers" : {
      "Content-Type" : [ "application/json" ],
      "Accept" : [ "application/json" ],
      "X-Api-Key" : [ "test-api-key" ]
    },
    "body" : {
      "type": "JSON",
      "json": [ 
        {
          "content" : "\"heritage=external-dns,external-dns/owner=default,external-dns/resource=service/ingress-nginx/ingress-nginx-controller\"",
          "disabled" : false,
          "name" : "a-example.test.com",
          "type" : "TXT"
        } 
      ],
      "matchType": "ONLY_MATCHING_FIELDS"
    }
  },
  "httpResponse" : {
    "statusCode" : 201,
    "body" : [
                {
                  "id": "1234",
                  "name": "a-example.test.com",
                  "rootName": "a-example.test.com",
                  "type": "TXT",
                  "content": "\"heritage=external-dns,external-dns/owner=default,external-dns/resource=service/ingress-nginx/ingress-nginx-controller\"",
                  "changeDate": "2022-23-03T13:04:25.772Z",
                  "ttl": 3600,
                  "prio": 0,
                  "disabled": false
                }
             ]
  }
}

HTTP/1.1 201
[Captures]
createATXTRecordExpectationId : jsonpath "$[0].id"



# ----------------- WHEN -----------------

# annotate the servive of ingress-nginx-controller
PATCH {{k8s_api_url}}/api/v1/namespaces/ingress-nginx/services/ingress-nginx-controller
Content-Type: application/merge-patch+json

{
  "metadata": {
    "annotations": {
      "external-dns.alpha.kubernetes.io/internal-hostname": "example.test.com"
    }
  }
}

HTTP/1.1 200

# ----------------- THEN -----------------

# get zones is called at least once
PUT {{dns_mockserver_url}}/mockserver/verify
Content-Type : application/json
[Options]
#very-verbose: true
retry: true
retry-interval: 6000
retry-max-count: 10

{
    "expectationId": {
      "id": "{{ getZonesExpectationId }}"
    },
    "times": {
        "atLeast": 1
    }
}

HTTP/1.1 202

# get zone is called at least once
PUT {{dns_mockserver_url}}/mockserver/verify
Content-Type : application/json
[Options]
#very-verbose: true
retry: true
retry-interval: 1000
retry-max-count: 3

{
    "expectationId": {
      "id": "{{ getZoneExpectationId }}"
    },
    "times": {
        "atLeast": 1
    }
}

HTTP/1.1 202

# create A record is called exactly once
PUT {{dns_mockserver_url}}/mockserver/verify
Content-Type : application/json
[Options]
#very-verbose: trues
retry: true
retry-interval: 1000
retry-max-count: 3

{
    "expectationId": {
      "id": "{{ createARecordExpectationId }}"
    },
    "times": {
        "atLeast": 1,
        "atMost": 1
    }
}

HTTP/1.1 202

# create TXT record is called exactly once
PUT {{dns_mockserver_url}}/mockserver/verify
Content-Type : application/json
[Options]
#very-verbose: trues
retry: true
retry-interval: 1000
retry-max-count: 3

{
    "expectationId": {
      "id": "{{ createTXTRecordExpectationId }}"
    },
    "times": {
        "atLeast": 1,
        "atMost": 1
    }
}

HTTP/1.1 202

# create TXT record is called exactly once
PUT {{dns_mockserver_url}}/mockserver/verify
Content-Type : application/json
[Options]
#very-verbose: trues
retry: true
retry-interval: 1000
retry-max-count: 3

{
    "expectationId": {
      "id": "{{ createATXTRecordExpectationId }}"
    },
    "times": {
        "atLeast": 1,
        "atMost": 1
    }
}

HTTP/1.1 202