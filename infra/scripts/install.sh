function certmanager() {
   helm upgrade --install certmanager ./infra/certmanager \
       --kubeconfig "${KUBECONFIG}" \
       --namespace certmanager \
       --create-namespace \
       --set-string docker.pullSecret="${DOCKER_PULL_SECRET}" \
       --set-string image.repository="${DOCKER_REPOSITORY}" \
       --set-string image.tag="${DOCKER_TAG}" \
       --set-string rev_id="${REV_ID}" \
       --set-file challenge.domains=./infra/certmanager/files/config.json
       -f ./infra/certmanager/values.yaml
}

"$@"