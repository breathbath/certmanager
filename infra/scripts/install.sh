function certmanager() {
   helm upgrade --install certmanager ./infra/certmanager \
       --kubeconfig "${KUBECONFIG}" \
       --namespace certmanager \
       --create-namespace
       --set docker.pullSecret="${DOCKER_PULL_SECRET}" \
       --set image.repository="${DOCKER_REPOSITORY}" \
       --set image.tag="${DOCKER_TAG}" \
       --set rev_id="${REV_ID}" \
       -f ./infra/certmanager/values.yaml
}

"$@"