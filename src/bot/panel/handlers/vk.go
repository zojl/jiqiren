package handlers

import "github.com/SevereCloud/vksdk/v2/api"

// vk is set once at startup (see panel.SetVK) so handlers can look up live
// VK profile data (e.g. manager names) without threading the client through
// every constructor.
var vk *api.VK

func SetVK(client *api.VK) {
	vk = client
}
