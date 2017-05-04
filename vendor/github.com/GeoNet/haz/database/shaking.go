package database

import "time"

/*
SavePGAVertical saves vertical Peak Ground Acceleration values to the db.
Source should be unique in a population.  A single value is stored for each
source.  The stored pga and time is updated if the new pga is greater.
*/
func (db *DB) SavePGAVertical(source string, longitude, latitude float64, t time.Time, pga float64) error {
	_, err := db.Exec("select impact.add_pga_vertical($1, $2, $3, $4, $5)",
		source, longitude, latitude, t, pga)

	return err
}

/*
SavePGAHorizontal saves horizontal Peak Ground Acceleration values to the db.
Source should be unique in a population.  A single value is stored for each
source.  The stored pga and time is updated if the new pga is greater.
*/
func (db *DB) SavePGAHorizontal(source string, longitude, latitude float64, t time.Time, pga float64) error {

	_, err := db.Exec("select impact.add_pga_horizontal($1, $2, $3, $4, $5)",
		source, longitude, latitude, t, pga)

	return err
}

/*
SavePGAVertical saves vertical Peak Ground Velocity values to the db.
Source should be unique in a population.  A single value is stored for each
source.  The stored pgv and time is updated if the new pgv is greater.
*/
func (db *DB) SavePGVVertical(source string, longitude, latitude float64, t time.Time, pgv float64) error {
	_, err := db.Exec("select impact.add_pgv_vertical($1, $2, $3, $4, $5)",
		source, longitude, latitude, t, pgv)

	return err
}

/*
SavePGAHorizontal saves horizontal Peak Ground Velocity values to the db.
Source should be unique in a population.  A single value is stored for each
source.  The stored pgv and time is updated if the new pgv is greater.
*/
func (db *DB) SavePGVHorizontal(source string, longitude, latitude float64, t time.Time, pgv float64) error {
	_, err := db.Exec("select impact.add_pgv_horizontal($1, $2, $3, $4, $5)",
		source, longitude, latitude, t, pgv)

	return err
}

/*
ZeroPGAVertical sets vertical Peak Ground Acceleration values that are older than ago
to 0.0
*/
func (db *DB) ZeroPGAVertical(ago time.Time) error {
	_, err := db.Exec(`UPDATE impact.pga SET vertical = 0.0 WHERE time_v < $1`, ago)

	return err
}

/*
ZeroPGAHorizontal sets horizontal Peak Ground Acceleration values that are older than ago
to 0.0
*/
func (db *DB) ZeroPGAHorizontal(ago time.Time) error {
	_, err := db.Exec(`UPDATE impact.pga SET horizontal = 0.0 WHERE time_v < $1`, ago)

	return err
}

/*
ZeroPGVVertical sets vertical Peak Ground Velocity values that are older than ago
to 0.0
*/
func (db *DB) ZeroPGVVertical(ago time.Time) error {
	_, err := db.Exec(`UPDATE impact.pgv SET vertical = 0.0 WHERE time_h < $1`, ago)

	return err
}

/*
ZeroPGVHorizontal sets vertical Peak Ground Velocity values that are older than ago
to 0.0
*/
func (db *DB) ZeroPGVHorizontal(ago time.Time) error {
	_, err := db.Exec(`UPDATE impact.pgv SET horizontal = 0.0 WHERE time_h < $1`, ago)

	return err
}

/*
DeletePGA deletes Peak Ground Acceleration values where times for both horizontal and vertical
values are older then ago.
*/
func (db *DB) DeletePGA(ago time.Time) error {
	_, err := db.Exec(`DELETE FROM impact.pga WHERE time_h < $1 AND time_v < $1`, ago)

	return err
}

/*
DeletePGV deletes Peak Ground Velocity values where times for both horizontal and vertical
values are older then ago.
*/
func (db *DB) DeletePGV(ago time.Time) error {
	_, err := db.Exec(`DELETE FROM impact.pgv WHERE time_h < $1 AND time_v < $1`, ago)

	return err
}