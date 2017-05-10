DROP SCHEMA IF EXISTS fdsn CASCADE;

CREATE SCHEMA fdsn;

CREATE TABLE fdsn.event (
  PublicID              TEXT                        NOT NULL PRIMARY KEY,
  ModificationTime      TIMESTAMP(6) WITH TIME ZONE NOT NULL,
  OriginTime            TIMESTAMP(6) WITH TIME ZONE NOT NULL,
  Latitude              NUMERIC                     NOT NULL,
  Longitude             NUMERIC                     NOT NULL,
  Depth                 NUMERIC                     NOT NULL,
  Magnitude             NUMERIC                     NOT NULL,
  MagnitudeType         TEXT                        NOT NULL,
  Deleted               BOOLEAN                     NOT NULL DEFAULT FALSE,
  EventType             TEXT                        NOT NULL,
  DepthType             TEXT                        NOT NULL,
  EvaluationMethod      TEXT                        NOT NULL,
  EarthModel            TEXT                        NOT NULL,
  EvaluationMode        TEXT                        NOT NULL,
  EvaluationStatus      TEXT                        NOT NULL,
  UsedPhaseCount        INTEGER                     NOT NULL,
  UsedStationCount      INTEGER                     NOT NULL,
  OriginError           NUMERIC                     NOT NULL,
  AzimuthalGap          NUMERIC                     NOT NULL,
  MinimumDistance       NUMERIC                     NOT NULL,
  MagnitudeUncertainty  NUMERIC                     NOT NULL,
  MagnitudeStationCount INTEGER                     NOT NULL,
  Origin_geom           GEOGRAPHY(POINT, 4326)      NOT NULL,
  Quakeml12Event        TEXT                        NOT NULL,
  Sc3ml                 TEXT                        NOT NULL
);

CREATE TABLE fdsn.stream (
  streamPK SERIAL PRIMARY KEY,
  network  TEXT NOT NULL,
  station  TEXT NOT NULL,
  channel  TEXT NOT NULL,
  location TEXT NOT NULL,
  UNIQUE (network, station, channel, location)
);

-- Table for index to the miniSEED files in the S3 bucket.
CREATE TABLE fdsn.holdings (
  streamPK INTEGER REFERENCES fdsn.stream (streamPK) ON DELETE CASCADE NOT NULL,
  start_time    TIMESTAMP(6) WITH TIME ZONE NOT NULL,
  numsamples INTEGER NOT NULL,
  key      TEXT                     NOT NULL,
  UNIQUE (streamPK, key)
);

CREATE FUNCTION fdsn.event_geom()
  RETURNS TRIGGER AS
$$
BEGIN
  NEW.origin_geom = ST_GeogFromWKB(st_AsEWKB(st_setsrid(st_makepoint(NEW.longitude, NEW.latitude), 4326)));
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER quake_geom_trigger BEFORE INSERT OR UPDATE ON fdsn.event
FOR EACH ROW EXECUTE PROCEDURE fdsn.event_geom();

CREATE INDEX ON fdsn.event (PublicID);
CREATE INDEX ON fdsn.event (ModificationTime);
CREATE INDEX ON fdsn.event (OriginTime);
CREATE INDEX ON fdsn.event (Magnitude);
CREATE INDEX ON fdsn.event (Depth);
CREATE INDEX ON fdsn.event (Latitude);
CREATE INDEX ON fdsn.event (Longitude);

CREATE OR REPLACE VIEW fdsn.quake_search_v1
AS
  SELECT
    publicID,
    eventType,
    originTime,
    modificationTime,
    latitude,
    longitude,
    depth,
    magnitude,
    evaluationMethod,
    evaluationStatus,
    evaluationMode,
    earthModel,
    depthType,
    originError,
    usedPhaseCount,
    usedStationCount,
    minimumDistance,
    azimuthalGap,
    magnitudeType,
    magnitudeUncertainty,
    magnitudeStationCount,
    origin_geom :: GEOMETRY
  FROM fdsn.event
  WHERE Deleted != TRUE
  ORDER BY OriginTime DESC;

--  for miniSEED records from SEEDLink

CREATE TABLE fdsn.record (
  streamPK   INTEGER REFERENCES fdsn.stream (streamPK) ON DELETE CASCADE NOT NULL,
  start_time TIMESTAMP(6) WITH TIME ZONE                                  NOT NULL,
  latency    NUMERIC                                                      NOT NULL,
  raw        BYTEA                                                        NOT NULL,
  PRIMARY KEY (streamPK, start_time)
);

CREATE INDEX ON fdsn.record (start_time);

