package api

// FIXME: this is a temporary hack that will only work with one city
/*
var (
	routeCache        []byte
	routeETag         string
	routeCacheControl string

	// one week
	routeAge = 60 * 60 * 24 * 7
)

type routesResp struct {
	Routes []*models.Route `json:"routes"`
}

func InitRouteCache() error {
	// getAll subway/train routes so we can pre-render them. Including
	// buses would be too much
	routes, err := models.GetPreloadRoutes(etc.DBConn, conf.Partner.AgencyIDs)
	if err != nil {
		log.Println("can't get routes", err)
		return err
	}

	rr := routesResp{
		Routes: routes,
	}

	b, err := json.Marshal(rr)
	if err != nil {
		log.Println("can't marshal routes", err)
		return err
	}

	routeCache = b
	routeETag = fmt.Sprintf("%x", md5.Sum(routeCache))
	routeCacheControl = fmt.Sprintf("max-age=%d", routeAge)

	return nil
}

func getRoutes(w http.ResponseWriter, r *http.Request) {
	var err error

	if len(routeCache) < 0 {
		log.Println("routeCache should be initialized before API is started")
		err = InitRouteCache()
		if err != nil {
			log.Println("couldn't init route cache", err)
			apiErr(w, err)
			return
		}
	}

	w.Header().Set("ETag", routeETag)
	w.Header().Set("Cache-Control", routeCacheControl)

	w.Write(routeCache)
}
*/
