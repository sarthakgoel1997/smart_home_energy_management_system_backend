package model

type Location struct {
	Id            uint32
	UnitNumber    uint32
	Street        uint32
	City          string
	State         string
	Zipcode       uint32
	Country       string
	SquareFootage float32
	BedroomsCount uint32
}

type ServiceLocation struct {
	Id             uint32
	CustomerId     uint32
	DateTakenOver  string
	OccupantsCount uint32
	UnitNumber     uint32
	Street         uint32
	City           string
	State          string
	Zipcode        uint32
	Country        string
	SquareFootage  float32
	BedroomsCount  uint32
	LocationLabel  string
	Active         uint32
}

type GetServiceLocationsResponse struct {
	ServiceLocations []ServiceLocation
}
