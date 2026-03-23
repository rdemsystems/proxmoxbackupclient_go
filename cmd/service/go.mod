module github.com/tizbac/proxmoxbackupclient_go/cmd/service

go 1.25

require (
	github.com/kardianos/service v1.2.2
	github.com/tizbac/proxmoxbackupclient_go/gui v0.0.0
)

replace github.com/tizbac/proxmoxbackupclient_go/gui => ../../gui
