wrk.method = "POST"
wrk.body   = "{\"uid\": 123123, \"content\":\"压测123123\",\"retries\":2}"
wrk.headers["Content-Type"] = "application/json"
