module "k8splay"

go 1.15

require (
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96 // indirect
	github.com/gin-gonic/gin v1.6.3
	github.com/gorilla/websocket v1.5.0
	github.com/shenyisyn/goft-gin v0.5.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/metrics v0.20.2
	sigs.k8s.io/yaml v1.2.0
)
