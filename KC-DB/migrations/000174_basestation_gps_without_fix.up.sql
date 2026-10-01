-- A base station without a position fix reports geoLocation [0,0,0]. Before
-- the service center recognized that report it stored 0/0 as a GPS position;
-- such a station keeps its GPS source without coordinates, which the web UI
-- shows as reporting no fix. A manual location is never touched.
UPDATE basestations
   SET latitude = NULL, longitude = NULL, altitude = NULL
 WHERE location_source = 'gps'
   AND latitude = 0
   AND longitude = 0;
