package database

import (
	"fmt"
	"github.com/GeoNet/haz/msg"
	"github.com/lib/pq"
	"log"
)

// These regions must exist in the DB.
var regionIDs = []msg.RegionID{
	msg.NewZealand,
}

func (db *DB) SaveQuake(q msg.Quake) error {

	//  could build the map from the struct using https://github.com/fatih/structs (reflection)
	//
	// Geom is added with a DB trigger for each new row
	var qv = map[string]interface{}{
		`PublicID`:              q.PublicID,
		`Type`:                  q.Type,
		`AgencyID`:              q.AgencyID,
		`ModificationTime`:      q.ModificationTime,
		`Time`:                  q.Time,
		`Longitude`:             q.Longitude,
		`Latitude`:              q.Latitude,
		`Depth`:                 q.Depth,
		`DepthType`:             q.DepthType,
		`MethodID`:              q.MethodID,
		`EarthModelID`:          q.EarthModelID,
		`EvaluationMode`:        q.EvaluationMode,
		`EvaluationStatus`:      q.EvaluationStatus,
		`UsedPhaseCount`:        q.UsedPhaseCount,
		`UsedStationCount`:      q.UsedStationCount,
		`StandardError`:         q.StandardError,
		`AzimuthalGap`:          q.AzimuthalGap,
		`MinimumDistance`:       q.MinimumDistance,
		`Magnitude`:             q.Magnitude,
		`MagnitudeUncertainty`:  q.MagnitudeUncertainty,
		`MagnitudeType`:         q.MagnitudeType,
		`MagnitudeStationCount`: q.MagnitudeStationCount,
		`Site`:                  q.Site,
		`Status`:                q.Status(),
		`Quality`:               q.Quality(),
		`Deleted`:               q.Status() == `deleted`,
		`BackupSite`:            q.Site == `backup`,
		`MMI`:                   q.MMI(),
	}

	mmi := q.MMI()
	qv[`Intensity`] = msg.MMIIntensity(mmi)
	qv[`MMI`] = int(mmi)

	// don't use time.UnixNano() for modificationTimeMicro, the zero time overflows int64.
	mtUnixMicro := q.ModificationTime.Unix()*1000000 + int64(q.ModificationTime.Nanosecond()/1000)
	qv[`ModificationTimeUnixMicro`] = mtUnixMicro

	// Add the region MMID and intensity for all regions in the DB.
	for _, v := range regionIDs {
		l, err := q.ClosestInRegion(v)
		if err != nil {
			log.Println("error finding closest locality in " + string(v))
			log.Println("setting MMID and intensity unknown.")
			qv[`MMID_`+string(v)] = 0.0
			qv[`Intensity_`+string(v)] = msg.MMIIntensity(0.0)
			continue
		}
		qv[`MMID_`+string(v)] = int(l.MMIDistance)
		qv[`Intensity_`+string(v)] = msg.MMIIntensity(l.MMIDistance)
	}

	var insert string
	var params string
	var values []interface{}
	var i int = 1

	for k, v := range qv {
		insert = insert + k + `, `
		params = params + fmt.Sprintf("$%d, ", i)
		values = append(values, v)
		i = i + 1
	}

	locality := "'unknown'"
	c, err := q.ClosestInRegion(msg.NewZealand)
	if err == nil {
		locality = fmt.Sprintf("$$%s$$", c.Location())
	}
	insert = insert + `Locality`
	params = params + locality

	// Quake History
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM haz.quakehistory WHERE PublicID = $1 AND ModificationTimeUnixMicro = $2`, q.PublicID, mtUnixMicro)
	if err != nil {
		txn.Rollback()
		return err
	}

	_, err = txn.Exec(`INSERT INTO haz.quakehistory(`+insert+`) VALUES( `+params+` )`, values...)
	if err != nil {
		txn.Rollback()
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	// Clean out old quake history
	_, err = db.Exec(`DELETE FROM haz.quakehistory WHERE time < now() - interval '365 days'`)

	// Quake
	txn, err = db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM haz.quake WHERE PublicID = $1 AND ModificationTime <= $2`, q.PublicID, q.ModificationTime)
	if err != nil {
		txn.Rollback()
		return err
	}

	_, err = txn.Exec(`INSERT INTO haz.quake(`+insert+`) VALUES( `+params+` )`, values...)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			// a unique_violation means the new quake info is older than in the table already.
			// this is not an error for this application - we want the latest information only in
			// the quake table.
			// http://www.postgresql.org/docs/9.3/static/errcodes-appendix.html
			if err.Code == `23505` {
				txn.Rollback()
				err = nil
			} else {
				txn.Rollback()
				return err
			}
		} else {
			txn.Rollback()
			return err
		}
	} else {
		err = txn.Commit()
		if err != nil {
			return err
		}
	}

	// Quake api
	txn, err = db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM haz.quakeapi WHERE PublicID = $1 AND ModificationTime <= $2`, q.PublicID, q.ModificationTime)
	if err != nil {
		txn.Rollback()
		return err
	}

	_, err = txn.Exec(`INSERT INTO haz.quakeapi(`+insert+`) VALUES( `+params+` )`, values...)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			// a unique_violation means the new quake info is older than in the table already.
			// this is not an error for this application - we want the latest information only in
			// the quake table.
			// http://www.postgresql.org/docs/9.3/static/errcodes-appendix.html
			if err.Code == `23505` {
				txn.Rollback()
				err = nil
			} else {
				txn.Rollback()
				return err
			}
		} else {
			txn.Rollback()
			return err
		}
	} else {
		err = txn.Commit()
		if err != nil {
			return err
		}
	}

	// Clean out old quakes from quakeapi
	_, err = db.Exec(`DELETE FROM haz.quakeapi WHERE time < now() - interval '365 days' OR status = 'duplicate'`)

	return err
}

func (db *DB) SaveHeartBeat(h msg.HeartBeat) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM haz.soh WHERE serverID = $1`, h.ServiceID)
	if err != nil {
		txn.Rollback()
		return err
	}

	_, err = txn.Exec(`INSERT INTO haz.soh(serverID, timeReceived) VALUES($1,$2)`, h.ServiceID, h.SentTime)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// SaveHeartBeatQRT saves heartbeat messages into the QRT schema.  This is used by
// the origin web servers.  This func can be removed once the web site is
// using the API.
func (db *DB) SaveHeartBeatQRT(h msg.HeartBeat) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM qrt.soh WHERE serverID = $1`, h.ServiceID)
	if err != nil {
		txn.Rollback()
		return err
	}

	_, err = txn.Exec(`INSERT INTO qrt.soh(serverID, timeReceived) VALUES($1,$2)`, h.ServiceID, h.SentTime)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// SaveHeartBeatQRT saves quake messages into the QRT schema.  This is used by
// the origin web servers.  This func can be removed once the web site is
// using the API.
func (db *DB) SaveQuakeQRT(q msg.Quake) error {
	_,err := db.Exec(`SELECT qrt.add_event($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
	q.PublicID, q.AgencyID, q.Latitude, q.Longitude, q.Time, q.ModificationTime, q.Depth,
	q.UsedPhaseCount, q.Magnitude, q.MagnitudeType, q.MagnitudeStationCount, q.Status(), q.Type)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO qrt.eventhistory(publicid, latitude, longitude, origintime,
	updatetime, depth, usedPhaseCount, magnitude, magnitudetype, magnitudeStationCount, status, type)
        VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
	q.PublicID, q.Latitude, q.Longitude, q.Time, q.ModificationTime, q.Depth, q.UsedPhaseCount, q.Magnitude,
	q.MagnitudeType, q.MagnitudeStationCount, q.Status(), q.Type)

	return err
}

