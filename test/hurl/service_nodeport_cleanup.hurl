# --- CLEANUP ---

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

