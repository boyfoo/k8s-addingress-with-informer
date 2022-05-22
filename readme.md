创建`service`时，带`annotation: ingress/http: true`注释的，自动生成ingress

### 部署

docker编译和上传：

`docker build -t registry.cn-hangzhou.aliyuncs.com/boyfoo/ingress-manager:1.0.0 .`

`docker push registry.cn-hangzhou.aliyuncs.com/boyfoo/ingress-manager:1.0.0`

生成yaml：`k create deployment ingress-manager --image registry.cn-hangzhou.aliyuncs.com/boyfoo/ingress-manager:1.0.0 --dry-run=client -o yaml > yaml/ingress-manager.yaml
`

此时虽然可以运行，但是因为容器内的角色没有权限所以是无法操作相关资源的，创建和绑定相关角色

`k apply -f ingress-manager-sa.yaml`