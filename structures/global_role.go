package structures

type GlobalRole int32

const (
	// Normal role has no extra permissions
	GlobalRoleUser GlobalRole = iota
	// Streamer role can go live and stream
	GlobalRoleStreamer GlobalRole = 100
	// Staff role has the permissions of admin in every channel and can also modify users and grant users the streamer role
	GlobalRoleStaff GlobalRole = 900
	// Owner role same as staff but can modify staff
	GlobalRoleOwner GlobalRole = 1000
)
