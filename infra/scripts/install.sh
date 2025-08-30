function certmanager() {
   helm upgrade --install certmanager ./infra/certmanager \
       --kubeconfig "${KUBECONFIG}" \
       --namespace certmanager \
       --create-namespace \
       --set-string docker.pullSecret="${DOCKER_PULL_SECRET}" \
       --set-string image.repository="${DOCKER_REPOSITORY}" \
       --set-string configPath="${CONFIG_PATH}" \
       --set-string image.tag="${DOCKER_TAG}" \
       --set-string rev_id="${REV_ID}" \
       -f ./infra/certmanager/values.yaml
}

function certmanagerConfig() {
   helm upgrade certmanager ./infra/certmanager \
       --reuse-values \
       --kubeconfig "${KUBECONFIG}" \
       --namespace certmanager \
       --set-string rev_id="${REV_ID}"
}

"$@"