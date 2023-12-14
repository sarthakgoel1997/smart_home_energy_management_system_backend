package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"shems/model"
	redisService "shems/redis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func LoginUser(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req model.LoginUserRequest
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rows, err := db.Query("SELECT * FROM Customers WHERE email = ?", req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var customer model.Customer
	for rows.Next() {
		err = rows.Scan(&customer.Id, &customer.FirstName, &customer.LastName, &customer.PhoneNumber, &customer.Email, &customer.BillingAddressId, &customer.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if customer.Id == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "User does not exist"})
		return
	}

	passwordMatches := CheckPasswordHash(req.Password, customer.Password)

	if !passwordMatches {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid password"})
		return
	}

	// Return customer details in response
	customer.Password = ""
	resp := model.LoginUserResponse{
		CustomerDetails: customer,
	}
	json.NewEncoder(w).Encode(resp)
}

func RegisterUser(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	var req model.RegisterUserRequest
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if user with same email already exists
	rows, err := db.Query("SELECT id FROM Customers WHERE email = ?", req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var customerId uint32
	for rows.Next() {
		err = rows.Scan(&customerId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if customerId > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "User with same email already exists"})
		return
	}

	// Check if same location already exists
	rows, err = db.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locationId uint32
	for rows.Next() {
		err = rows.Scan(&locationId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create new location if this does not exist
	if locationId == 0 {
		// Create location
		_, err = db.Exec("INSERT INTO Locations (unit_number, street, city, state, zipcode, country, square_footage, bedrooms_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country, req.SquareFootage, req.BedroomsCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get location id
		rows, err = db.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&locationId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	passWordHash, _ := GetPasswordHash(req.Password)

	// Create user
	_, err = db.Exec("INSERT INTO Customers (first_name, last_name, phone_number, email, billing_address_id, password) VALUES (?, ?, ?, ?, ?, ?)", req.FirstName, req.LastName, req.PhoneNumber, req.Email, locationId, passWordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user
	rows, err = db.Query("SELECT * FROM Customers WHERE email = ?", req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var customer model.Customer
	for rows.Next() {
		err = rows.Scan(&customer.Id, &customer.FirstName, &customer.LastName, &customer.PhoneNumber, &customer.Email, &customer.BillingAddressId, &customer.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Return customer details in response
	customer.Password = ""
	resp := model.LoginUserResponse{
		CustomerDetails: customer,
	}
	json.NewEncoder(w).Encode(resp)
}

func GetDashboardData(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// Get customer id from query params
	var customerIdInt int
	customerIdStr := r.URL.Query().Get("customerId")
	if len(customerIdStr) == 0 {
		http.Error(w, "Customer Id cannot be empty", http.StatusBadRequest)
		return
	}
	customerIdInt, err := strconv.Atoi(customerIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if customerIdInt == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	currentDate := r.URL.Query().Get("currentDate")
	startDateTime, _ := GetStartOfMonth(currentDate)

	startOfNextMonth := time.Date(startDateTime.Year(), startDateTime.Month()+1, 1, 0, 0, 0, 0, startDateTime.Location())
	endDateTime := startOfNextMonth.Add(-time.Second)

	// get energy consumption and costs by service locations
	query := queryToFetchEnergyCostsByServiceLocations()
	rows, err := db.Query(query, startDateTime, endDateTime, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var serviceLocationCosts []model.ServiceLocationCost
	var slc model.ServiceLocationCost
	var totalEnergyCost, totalEnergyConsumption float32

	for rows.Next() {
		err = rows.Scan(&slc.LocationId, &slc.UnitNumber, &slc.Street, &slc.City, &slc.Zipcode, &slc.State, &slc.Country, &slc.EnergyConsumption, &slc.EnergyCost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		serviceLocationCosts = append(serviceLocationCosts, slc)
		totalEnergyCost = float32(totalEnergyCost + slc.EnergyCost)
		totalEnergyConsumption = float32(totalEnergyConsumption + slc.EnergyConsumption)
	}

	// get average energy consumption for similar locations
	query = queryForAverageEnergyConsumptionForSimilarServiceLocations()
	rows, err = db.Query(query, startDateTime, endDateTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	similarLocationAvgEnergyConsumption := make(map[uint32]float32)
	var locationId uint32
	var avgEnergyConsumption float32
	for rows.Next() {
		err = rows.Scan(&locationId, &avgEnergyConsumption)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		similarLocationAvgEnergyConsumption[locationId] = avgEnergyConsumption
	}
	for i := range serviceLocationCosts {
		serviceLocationCosts[i].SimilarLocationsAverageEnergyConsumption = similarLocationAvgEnergyConsumption[serviceLocationCosts[i].LocationId]
	}

	// get hourly prices
	query = queryToGetHourlyPrices()
	rows, err = db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var hourlyPrices []model.Price
	var p model.Price
	for rows.Next() {
		err = rows.Scan(&p.Zipcode, &p.Hour, &p.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		hourlyPrices = append(hourlyPrices, p)
	}

	// get energy consumption by devices
	query = queryToFetchEnergyConsumptionByDevices()
	rows, err = db.Query(query, startDateTime, endDateTime, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var enrolledDevices []model.EnrolledDevicesEnergyConsumption
	for rows.Next() {
		var edId uint32
		var dType, dModelNumber, edAliasName string
		var energyConsumption float32

		err = rows.Scan(&edId, &dType, &dModelNumber, &edAliasName, &energyConsumption)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		enrolledDevices = append(enrolledDevices, model.EnrolledDevicesEnergyConsumption{
			DeviceLabel:       fmt.Sprint(edAliasName, " (", dType, " - ", dModelNumber, ")"),
			EnergyConsumption: energyConsumption,
		})
	}

	resp := model.DashboardDataResponse{
		ServiceLocationCosts:   serviceLocationCosts,
		TotalEnergyCost:        totalEnergyCost,
		HourlyPrices:           hourlyPrices,
		EnrolledDevices:        enrolledDevices,
		TotalEnergyConsumption: totalEnergyConsumption,
	}
	json.NewEncoder(w).Encode(resp)
}

func GetEnrolledDevices(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// Get customer id from query params
	var customerIdInt int
	customerIdStr := r.URL.Query().Get("customerId")
	if len(customerIdStr) == 0 {
		http.Error(w, "Customer Id cannot be empty", http.StatusBadRequest)
		return
	}
	customerIdInt, err := strconv.Atoi(customerIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if customerIdInt == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	// get enrolled devices
	query := queryToGetEnrolledDevices()
	rows, err := db.Query(query, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var enrolledDevices []model.EnrolledDevice
	var ed model.EnrolledDevice
	for rows.Next() {
		err = rows.Scan(&ed.Id, &ed.ServiceLocationId, &ed.DeviceId, &ed.AliasName, &ed.RoomNumber, &ed.Active)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		enrolledDevices = append(enrolledDevices, ed)
	}

	// get all devices
	query = queryToGetAllDevices()
	rows, err = db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []model.Device
	var d model.Device
	for rows.Next() {
		err = rows.Scan(&d.Id, &d.Type, &d.ModelNumber)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		devices = append(devices, d)
	}

	// get all service locations
	query = queryToGetAllServiceLocations()
	rows, err = db.Query(query, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var serviceLocations []model.ServiceLocation
	var sl model.ServiceLocation
	for rows.Next() {
		err = rows.Scan(&sl.Id, &sl.CustomerId, &sl.DateTakenOver, &sl.OccupantsCount, &sl.UnitNumber, &sl.Street, &sl.City, &sl.State, &sl.Zipcode, &sl.Country, &sl.SquareFootage, &sl.BedroomsCount, &sl.Active)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		serviceLocations = append(serviceLocations, sl)
	}

	// Map service location and device names in enrolledDevices array
	serviceLocationsMap := make(map[uint32]string)
	for i, sl := range serviceLocations {
		serviceLocationsMap[sl.Id] = fmt.Sprint(sl.UnitNumber, ", ", sl.Street, ", ", sl.City, ", ", sl.State, ", ", sl.Zipcode, ", ", sl.Country)
		serviceLocations[i].LocationLabel = fmt.Sprint(sl.UnitNumber, ", ", sl.Street, ", ", sl.City, ", ", sl.State, ", ", sl.Zipcode, ", ", sl.Country)
	}

	devicesMap := make(map[uint32]string)
	deviceTypesMap := make(map[uint32]string)
	for i, d := range devices {
		devicesMap[d.Id] = fmt.Sprint(d.ModelNumber, " (", d.Type, ")")
		deviceTypesMap[d.Id] = d.Type

		devices[i].DeviceName = devicesMap[d.Id]
	}

	for i := range enrolledDevices {
		enrolledDevices[i].ServiceLocation = serviceLocationsMap[enrolledDevices[i].ServiceLocationId]
		enrolledDevices[i].Device = devicesMap[enrolledDevices[i].DeviceId]
		enrolledDevices[i].DeviceType = deviceTypesMap[enrolledDevices[i].DeviceId]
	}

	resp := model.GetEnrolledDevicesResponse{
		EnrolledDevices:  enrolledDevices,
		ServiceLocations: serviceLocations,
		Devices:          devices,
	}
	json.NewEncoder(w).Encode(resp)
}

func DeleteEnrolledDevice(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	w.Header().Set("Content-Type", "application/json")

	// Get customer id from query params
	var customerIdInt int
	customerIdStr := r.URL.Query().Get("customerId")
	if len(customerIdStr) == 0 {
		http.Error(w, "Customer Id cannot be empty", http.StatusBadRequest)
		return
	}
	customerIdInt, err := strconv.Atoi(customerIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if customerIdInt == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	// Get enrolled device id from query params
	var enrolledDeviceIdInt int
	enrolledDeviceIdStr := r.URL.Query().Get("enrolledDeviceId")
	if len(enrolledDeviceIdStr) == 0 {
		http.Error(w, "Enrolled Device Id cannot be empty", http.StatusBadRequest)
		return
	}
	enrolledDeviceIdInt, err = strconv.Atoi(enrolledDeviceIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if enrolledDeviceIdInt == 0 {
		http.Error(w, "Enrolled Device Id cannot be 0", http.StatusBadRequest)
		return
	}

	// validation: check if enrolled device exists by same customer
	query := queryToCheckIfEnrolledDeviceExists()
	rows, err := db.Query(query, enrolledDeviceIdInt, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var checkId int
	for rows.Next() {
		err = rows.Scan(&checkId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if checkId != enrolledDeviceIdInt {
		http.Error(w, "Enrolled Device does not exist", http.StatusBadRequest)
		return
	}

	// delete enrolled device
	query = queryToDeleteEnrolledDevice()
	_, err = db.Exec(query, enrolledDeviceIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Enrolled device deleted successfully"))
}

func AddEnrolledDevice(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	var req model.EnrolledDevice
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	if req.CustomerId == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.ServiceLocationId == 0 {
		http.Error(w, "Service Location Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.DeviceId == 0 {
		http.Error(w, "Device Id cannot be 0", http.StatusBadRequest)
		return
	}

	// validation: check if service location exists by same customer
	query := queryToCheckIfServiceLocationExists()
	rows, err := db.Query(query, req.ServiceLocationId, req.CustomerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var checkId uint32
	for rows.Next() {
		err = rows.Scan(&checkId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if checkId != req.ServiceLocationId {
		http.Error(w, "Service Location does not exist", http.StatusBadRequest)
		return
	}

	redisKey := "AddEnrolledDevice_CustomerId_" + fmt.Sprint(req.CustomerId)
	defer func() {
		// delete redis key
		err = redisService.DeleteKey(ctx, redisClient, redisKey)
		if err != nil {
			fmt.Println("error while deleting redis key", err.Error())
		}
	}()

	// take redis lock to avoid concurrent access or double clicking
	redisErr := TakeRedisLock(ctx, redisClient, redisKey)
	if len(redisErr) > 0 {
		http.Error(w, redisErr, http.StatusBadRequest)
		return
	}

	// insert query
	query = queryToAddEnrolledDevice()
	_, err = db.Exec(query, req.ServiceLocationId, req.DeviceId, req.AliasName, req.RoomNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Device enrolled successfully"))
}

func UpdateEnrolledDevice(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	var req model.EnrolledDevice
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	if req.CustomerId == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.Id == 0 {
		http.Error(w, "Enrolled Device Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.ServiceLocationId == 0 {
		http.Error(w, "Service Location Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.DeviceId == 0 {
		http.Error(w, "Device Id cannot be 0", http.StatusBadRequest)
		return
	}

	redisKey := "UpdateEnrolledDevice_CustomerId_" + fmt.Sprint(req.CustomerId)
	defer func() {
		// delete redis key
		err = redisService.DeleteKey(ctx, redisClient, redisKey)
		if err != nil {
			fmt.Println("error while deleting redis key", err.Error())
		}
	}()

	// take redis lock to avoid concurrent access or double clicking
	redisErr := TakeRedisLock(ctx, redisClient, redisKey)
	if len(redisErr) > 0 {
		http.Error(w, redisErr, http.StatusBadRequest)
		return
	}

	// validation: check if enrolled device exists by same customer
	query := queryToCheckIfEnrolledDeviceExists()
	rows, err := db.Query(query, req.Id, req.CustomerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var checkId int
	for rows.Next() {
		err = rows.Scan(&checkId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if checkId != int(req.Id) {
		http.Error(w, "Enrolled Device does not exist", http.StatusBadRequest)
		return
	}

	// update enrolled device
	query = queryToUpdateEnrolledDevice()
	_, err = db.Exec(query, req.ServiceLocationId, req.DeviceId, req.AliasName, req.RoomNumber, req.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Enrolled device updated successfully"))
}

func GetServiceLocations(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// Get customer id from query params
	var customerIdInt int
	customerIdStr := r.URL.Query().Get("customerId")
	if len(customerIdStr) == 0 {
		http.Error(w, "Customer Id cannot be empty", http.StatusBadRequest)
		return
	}
	customerIdInt, err := strconv.Atoi(customerIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if customerIdInt == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	// get all service locations
	query := queryToGetAllServiceLocations()
	rows, err := db.Query(query, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var serviceLocations []model.ServiceLocation
	var sl model.ServiceLocation
	for rows.Next() {
		err = rows.Scan(&sl.Id, &sl.CustomerId, &sl.DateTakenOver, &sl.OccupantsCount, &sl.UnitNumber, &sl.Street, &sl.City, &sl.State, &sl.Zipcode, &sl.Country, &sl.SquareFootage, &sl.BedroomsCount, &sl.Active)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		serviceLocations = append(serviceLocations, sl)
	}

	resp := model.GetServiceLocationsResponse{
		ServiceLocations: serviceLocations,
	}
	json.NewEncoder(w).Encode(resp)
}

func DeleteServiceLocation(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	w.Header().Set("Content-Type", "application/json")

	// Get customer id from query params
	var customerIdInt int
	customerIdStr := r.URL.Query().Get("customerId")
	if len(customerIdStr) == 0 {
		http.Error(w, "Customer Id cannot be empty", http.StatusBadRequest)
		return
	}
	customerIdInt, err := strconv.Atoi(customerIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if customerIdInt == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	// Get service location id from query params
	var serviceLocationIdInt int
	serviceLocationIdStr := r.URL.Query().Get("serviceLocationId")
	if len(serviceLocationIdStr) == 0 {
		http.Error(w, "Service Location Id cannot be empty", http.StatusBadRequest)
		return
	}
	serviceLocationIdInt, err = strconv.Atoi(serviceLocationIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if serviceLocationIdInt == 0 {
		http.Error(w, "Service Location Id cannot be 0", http.StatusBadRequest)
		return
	}

	// validation: check if service location exists by same customer
	query := queryToCheckIfServiceLocationExists()
	rows, err := db.Query(query, serviceLocationIdInt, customerIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var checkId int
	for rows.Next() {
		err = rows.Scan(&checkId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if checkId != serviceLocationIdInt {
		http.Error(w, "Service Location does not exist", http.StatusBadRequest)
		return
	}

	// delete service location
	query = queryToDeleteServiceLocation()
	_, err = db.Exec(query, serviceLocationIdInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service location deleted successfully"))
}

func AddServiceLocation(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB, redisClient *redis.Client) {
	var req model.ServiceLocation
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	if req.CustomerId == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	redisKey := "AddServiceLocation_CustomerId_" + fmt.Sprint(req.CustomerId)
	rollback := true
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		// delete redis key
		err = redisService.DeleteKey(ctx, redisClient, redisKey)
		if err != nil {
			fmt.Println("error while deleting redis key", err.Error())
		}

		if rollback {
			tx.Rollback()
			fmt.Println("Transaction rolled back")
		} else {
			tx.Commit()
			fmt.Println("Transaction committed")
		}
	}()

	// take redis lock to avoid concurrent access or double clicking
	redisErr := TakeRedisLock(ctx, redisClient, redisKey)
	if len(redisErr) > 0 {
		http.Error(w, redisErr, http.StatusBadRequest)
		return
	}

	// Check if same location already exists in locations table
	rows, err := tx.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locationId uint32
	for rows.Next() {
		err = rows.Scan(&locationId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create new location if this does not exist
	if locationId == 0 {
		// Create location
		_, err = tx.Exec("INSERT INTO Locations (unit_number, street, city, state, zipcode, country, square_footage, bedrooms_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country, req.SquareFootage, req.BedroomsCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get location id
		rows, err = tx.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&locationId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// validation: check if service location exists by same customer
	query := queryToCheckIfServiceLocationExistsByLocationId()
	rows, err = tx.Query(query, locationId, req.CustomerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var serviceLocationId int
	for rows.Next() {
		err = rows.Scan(&serviceLocationId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if serviceLocationId > 0 {
		http.Error(w, "Service Location already exists", http.StatusBadRequest)
		return
	}

	// insert query to add service location
	query = queryToAddServiceLocation()
	_, err = tx.Exec(query, req.CustomerId, locationId, req.DateTakenOver, req.OccupantsCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rollback = false

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service location added successfully"))
}

func UpdateServiceLocation(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB, redisClient *redis.Client) {
	var req model.ServiceLocation
	w.Header().Set("Content-Type", "application/json")

	// Parse the incoming JSON data from the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	if req.Id == 0 {
		http.Error(w, "Service Location Id cannot be 0", http.StatusBadRequest)
		return
	}
	if req.CustomerId == 0 {
		http.Error(w, "Customer Id cannot be 0", http.StatusBadRequest)
		return
	}

	redisKey := "UpdateServiceLocation_CustomerId_" + fmt.Sprint(req.CustomerId)
	defer func() {
		// delete redis key
		err = redisService.DeleteKey(ctx, redisClient, redisKey)
		if err != nil {
			fmt.Println("error while deleting redis key", err.Error())
		}
	}()

	// take redis lock to avoid concurrent access or double clicking
	redisErr := TakeRedisLock(ctx, redisClient, redisKey)
	if len(redisErr) > 0 {
		http.Error(w, redisErr, http.StatusBadRequest)
		return
	}

	// Check if same location already exists in locations table
	rows, err := db.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locationId uint32
	for rows.Next() {
		err = rows.Scan(&locationId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create new location if this does not exist
	if locationId == 0 {
		// Create location
		_, err = db.Exec("INSERT INTO Locations (unit_number, street, city, state, zipcode, country, square_footage, bedrooms_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country, req.SquareFootage, req.BedroomsCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get location id
		rows, err = db.Query("SELECT id FROM Locations WHERE unit_number = ? AND street = ? AND city = ? AND state = ? AND zipcode = ? AND country = ?", req.UnitNumber, req.Street, req.City, req.State, req.Zipcode, req.Country)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&locationId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// validation: check if service location exists by same customer
	query := queryToCheckIfServiceLocationExistsByLocationId()
	rows, err = db.Query(query, locationId, req.CustomerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var serviceLocationId int
	for rows.Next() {
		err = rows.Scan(&serviceLocationId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if serviceLocationId > 0 && serviceLocationId != int(req.Id) {
		http.Error(w, "Service Location with same address already exists", http.StatusBadRequest)
		return
	}

	// validation: check if service location id in request exists by same customer
	query = queryToCheckIfServiceLocationExists()
	rows, err = db.Query(query, req.Id, req.CustomerId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var checkId int
	for rows.Next() {
		err = rows.Scan(&checkId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if checkId != int(req.Id) {
		http.Error(w, "Service location does not exist", http.StatusBadRequest)
		return
	}

	// update service location
	query = queryToUpdateServiceLocation()
	_, err = db.Exec(query, locationId, req.DateTakenOver, req.OccupantsCount, req.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with a success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Service location updated successfully"))
}
