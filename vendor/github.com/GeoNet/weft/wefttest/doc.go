/*
Package wefttest assists with integration testing web applications.

A minimal test for a web application would look like:

	// routes will be tested with defaults; method GET expecting http.StatusOK.
	var routes = Requests{
		{ID: L(), URL: "/path/to/test/one"},
		{ID: L(), URL: "/path/to/test/two"},
		{ID: L(), URL: "/path/to/test/three"},
	}

	// Test all routes give the expected response.  Also check with
	// cache busters and extra query parameters.
	func TestRoutes(t *testing.T) {
		setup(t)
		defer teardown()

		for _, r := range routes {
			if b, err := r.Do(testServer.URL); err != nil {
				t.Error(err)
				t.Error(string(b))
			}
		}

		if err := routes.DoCheckQuery(testServer.URL); err != nil {
			t.Error(err)
		}
	}

*/
package wefttest
