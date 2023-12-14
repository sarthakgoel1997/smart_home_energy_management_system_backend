package users

func queryToFetchEnergyCostsByServiceLocations() string {
	sqlQuery := `
		SELECT
			l.id AS location_id,
			l.unit_number,
			l.street,
			l.city,
			l.zipcode,
			l.state,
			l.country,
			SUM(CASE WHEN e.value IS NOT NULL THEN e.value ELSE 0 END) AS total_energy_consumption,
			SUM(CASE WHEN e.value IS NOT NULL THEN e.value * p.value ELSE 0 END) AS total_energy_cost
		FROM
			Service_Locations sl
		LEFT JOIN
			Locations l ON l.id = sl.location_id
		LEFT JOIN
			Enrolled_Devices ed ON ed.service_location_id = sl.id
		LEFT JOIN
			Events e ON e.enrolled_device_id = ed.id AND e.label = 'energy use' AND e.created_at >= ? AND e.created_at <= ?
		LEFT JOIN
			Prices p ON p.zipcode = l.zipcode AND p.hour = HOUR(e.created_at) + 1
		WHERE
			sl.customer_id = ?
		GROUP BY
			1, 2, 3, 4, 5, 6, 7;
	`
	return sqlQuery
}

func queryForAverageEnergyConsumptionForSimilarServiceLocations() string {
	sqlQuery := `
	SELECT
		l2.id AS location_id, 
		SUM(CASE WHEN e2.value IS NOT NULL THEN e2.value ELSE 0 END) / (CASE WHEN COUNT(DISTINCT l3.id) > 0 THEN COUNT(DISTINCT l3.id) ELSE 1 END) AS average_energy_consumption
	FROM
		Locations l2
	INNER JOIN
		Locations l3 ON l3.square_footage >= l2.square_footage * 0.95 AND l3.square_footage <= l2.square_footage * 1.05
	LEFT JOIN
		Service_Locations sl2 ON sl2.location_id = l3.id
	LEFT JOIN
		Enrolled_Devices ed2 ON ed2.service_location_id = sl2.id
	LEFT JOIN
		Events e2 ON e2.enrolled_device_id = ed2.id AND e2.label = 'energy use' AND e2.created_at >= ? AND e2.created_at <= ?
	GROUP BY
		l2.id;
	`
	return sqlQuery
}

func queryToGetEnrolledDevices() string {
	sqlQuery := `
	SELECT
		ed.*
	FROM
		Enrolled_Devices ed
	INNER JOIN
		Service_Locations sl ON sl.id = ed.service_location_id
	WHERE
		sl.customer_id = ?
	ORDER BY
		sl.id, ed.id;
	`
	return sqlQuery
}

func queryToGetAllDevices() string {
	sqlQuery := `
	SELECT
		*
	FROM
		Devices;
	`
	return sqlQuery
}

func queryToGetAllServiceLocations() string {
	sqlQuery := `
	SELECT
		sl.id, sl.customer_id, sl.date_taken_over, sl.occupants_count, l.unit_number, l.street, l.city, l.state, l.zipcode, l.country, l.square_footage, l.bedrooms_count, sl.active
	FROM
		service_locations sl
	INNER JOIN
		Locations l ON l.id = sl.location_id
	WHERE
		customer_id = ?;
	`
	return sqlQuery
}

func queryToCheckIfEnrolledDeviceExists() string {
	sqlQuery := `
	SELECT
		ed.id
	FROM
		Enrolled_Devices ed
	INNER JOIN
		Service_Locations sl ON ed.service_location_id = sl.id
	WHERE
		ed.id = ?
		AND sl.customer_id = ?;
	`
	return sqlQuery
}

func queryToAddEnrolledDevice() string {
	sqlQuery := `
				INSERT INTO Enrolled_Devices
					(service_location_id, device_id, alias_name, room_number)
				VALUES
					(?, ?, ?, ?);
				`
	return sqlQuery
}

func queryToUpdateEnrolledDevice() string {
	sqlQuery := `
				UPDATE
					Enrolled_Devices
				SET
					service_location_id = ?,
					device_id = ?,
					alias_name = ?,
					room_number = ?
				WHERE
					id = ?;
				`
	return sqlQuery
}

func queryToDeleteEnrolledDevice() string {
	sqlQuery := `
				Update
					Enrolled_Devices
				SET
					Active = 0
				WHERE
					id = ?;
				`
	return sqlQuery
}

func queryToCheckIfServiceLocationExists() string {
	sqlQuery := `
	SELECT
		id
	FROM
		Service_Locations
	WHERE
		id = ?
		AND customer_id = ?;
	`
	return sqlQuery
}

func queryToDeleteServiceLocation() string {
	sqlQuery := `
				UPDATE
					Service_Locations
				SET
					Active = 0
				WHERE
					id = ?;
				`
	return sqlQuery
}

func queryToAddServiceLocation() string {
	sqlQuery := `
				INSERT INTO Service_Locations
					(customer_id, location_id, date_taken_over, occupants_count)
				VALUES
					(?, ?, ?, ?);
				`
	return sqlQuery
}

func queryToCheckIfServiceLocationExistsByLocationId() string {
	sqlQuery := `
	SELECT
		Id
	FROM
		Service_Locations
	WHERE
		location_id = ?
		AND customer_id = ?;
	`
	return sqlQuery
}

func queryToUpdateServiceLocation() string {
	sqlQuery := `
				UPDATE
					Service_Locations
				SET
					location_id = ?,
					date_taken_over = ?,
					occupants_count = ?
				WHERE
					id = ?;
				`
	return sqlQuery
}

func queryToGetHourlyPrices() string {
	sqlQuery := "SELECT * FROM Prices ORDER BY zipcode, hour;"
	return sqlQuery
}

func queryToFetchEnergyConsumptionByDevices() string {
	sqlQuery := `
	SELECT
		ed.id AS enrolled_device_id,
		d.type,
		d.model_number,
		ed.alias_name,
		SUM(CASE WHEN e.value IS NOT NULL THEN e.value ELSE 0 END) AS total_energy_consumption
	FROM
		Enrolled_Devices ed
	INNER JOIN
		Service_Locations sl ON sl.id = ed.service_location_id
	INNER JOIN
		Devices d ON d.id = ed.device_id
	LEFT JOIN
		Events e ON e.enrolled_device_id = ed.id AND e.label = 'energy use' AND e.created_at >= ? AND e.created_at <= ?
	WHERE
		sl.customer_id = ?
	GROUP BY
		1, 2, 3, 4;
	`
	return sqlQuery
}
