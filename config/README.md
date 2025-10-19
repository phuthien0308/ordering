# Purpose 
The goal of this project is to provide a central configuration management tool which is backed by zookeeper as the data storage. As a result, running this project requires some setting-ups such as zookeeper and zookeeper-ui( it is used for management)  
There are two endpoints:  
- GET:
- WATCH:
## Set up
### Zookeeper  
The storage of this project. Currently, it is installed via several configuration files in the folder named zookeeper. This should be done in order.
```
kubectl apply -f zookeeper-pvc.yaml
kubectl apply -f zookeeper-service.yaml
kubectl apply -f zookeeper-deployment.yaml 

```

### Zookeeper UI
This is done by install helm chart  

```
helm install zookeeper-ui --namespace zookeeper-single lowess-helm/zoonavigator \
    --set zoonavigator.env.AUTO_CONNECT_CONNECTION_STRING=zookeeper-single.zookeeper-single:2181 \
    --set zoonavigator.env.CONNECTION_LOCALZK_NAME=localzookeeper \
    --set zoonavigator.env.CONNECTION_LOCALZK_CONN=zookeeper-single.zookeeper-single:2181
```
This is the zookeeper ui, which makes easier to manage zookeeper


