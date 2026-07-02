
minikube start --driver=docker
minikube image load keycloak/keycloak:latest
minikube image load  postgres:16
minikube image load  grpc-server:v5
minikube image load  grpc-client:v51
#minikube image load promethius:vv5
#minikube image load employee-service:vv5

#kubectl apply -f grpcdemo/grpcdemo/greeterpb/client/clientCM.yaml
#kubectl apply -f postgres_keyclock-deployment.yaml
#kubectl apply -f keycloack-deployment.yaml
#kubectl apply -f grpc-server-deployment.yaml
#kubectl apply -f grpc-client-deployemt.yaml
#kubectl apply -f  promethius/deployment.yaml
#kubectl apply -f promethius/serviceMonitor.yaml
#kubectl apply -f promethius/PromethiusService.yaml
#helm upgrade --install promethius-app ~/test/promethius/deployment/promethius-chart

#ubectl apply -f promethius/employee_service.yaml

#/home/pradeep/test/promethius/createPortForwards.sh

