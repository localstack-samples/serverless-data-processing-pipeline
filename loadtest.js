import http from 'k6/http';
import { check } from 'k6';

// Get the API_ENDPOINT environment variable.
const API_ENDPOINT = __ENV.API_ENDPOINT;
if (!API_ENDPOINT) {
    throw new Error("Please set the API_ENDPOINT environment variable");
}

export let options = {
    vus: 10, // Number of virtual users.
    duration: '60s', // Duration of the test.
};

export default function () {
    // Get current utc timestamp.
    const timestamp = Math.floor(Date.now() / 1000);
    // Create unique id for each request.
    const id = `${__VU}-${__ITER}`;

    // Create a JSON payload.
    const data = {
        id: id.toString(),
        message: "Inbound request",
        timestamp: timestamp,
    };
    const body = JSON.stringify(data);

    // Send a POST request to the server.
    let response = http.post(
        API_ENDPOINT,
        body,
        { headers: { 'Content-Type': 'application/json' } },
    );

    // Check the response.
    check(response, {
        'status was 200': (r) => r.status == 200,
        'transaction time OK': (r) => r.timings.duration < 4000,
    });

    // If the status code is not 200, print the status code and the body of the response.
    if (response.status !== 200) {
        console.log(`Response status: ${response.status}, body: ${response.body}`);
    }
}
