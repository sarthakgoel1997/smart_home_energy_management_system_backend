package model

type Device struct {
	Id          uint32
	Type        string
	ModelNumber string
	DeviceName  string
}

type EnrolledDevice struct {
	Id                uint32
	ServiceLocationId uint32
	DeviceId          uint32
	AliasName         string
	RoomNumber        uint32
	CustomerId        uint32
	ServiceLocation   string
	DeviceType        string
	Device            string
	Active            uint32
}

type GetEnrolledDevicesResponse struct {
	EnrolledDevices  []EnrolledDevice
	Devices          []Device
	ServiceLocations []ServiceLocation
}
