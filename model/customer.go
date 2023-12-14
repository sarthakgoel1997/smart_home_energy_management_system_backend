package model

type Customer struct {
	Id               uint32
	FirstName        string
	LastName         string
	PhoneNumber      string
	Email            string
	BillingAddressId uint32
	Password         string
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserResponse struct {
	CustomerDetails Customer
}

type RegisterUserRequest struct {
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	PhoneNumber   string  `json:"phoneNumber"`
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	UnitNumber    uint32  `json:"unitNumber"`
	Street        uint32  `json:"street"`
	City          string  `json:"city"`
	State         string  `json:"state"`
	Zipcode       uint32  `json:"zipcode"`
	Country       string  `json:"country"`
	SquareFootage float32 `json:"squareFootage"`
	BedroomsCount uint32  `json:"bedroomsCount"`
}

type RegisterUserResponse struct {
	CustomerDetails Customer
}

type ServiceLocationCost struct {
	LocationId                               uint32
	UnitNumber                               uint32
	Street                                   uint32
	City                                     string
	Zipcode                                  uint32
	State                                    string
	Country                                  string
	EnergyConsumption                        float32
	EnergyCost                               float32
	SimilarLocationsAverageEnergyConsumption float32
}

type EnrolledDevicesEnergyConsumption struct {
	DeviceLabel       string
	EnergyConsumption float32
}

type DashboardDataResponse struct {
	ServiceLocationCosts   []ServiceLocationCost
	TotalEnergyConsumption float32
	TotalEnergyCost        float32
	HourlyPrices           []Price
	EnrolledDevices        []EnrolledDevicesEnergyConsumption
}
