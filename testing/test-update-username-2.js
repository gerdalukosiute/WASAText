const fetch = require("node-fetch")

const {
  API_URL,
  ALICE_ID,
} = require('./details');

async function testUpdateUsername(newName) {
    console.log(`Testing update username to: ${newName}`);
    try {
        const response = await fetch(`${API_URL}/user`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'X-User-ID': ALICE_ID
            },
            body: JSON.stringify({ newName })
        });

        let data;
        const contentType = response.headers.get("content-type");
        if (contentType && contentType.indexOf("application/json") !== -1) {
            data = await response.json();
        } else {
            data = await response.text();
        }

        console.log('Status:', response.status);
        console.log('Response:', data);

        return { status: response.status, data };
    } catch (error) {
        console.error('Error:', error.message);
        return { status: 500, error: error.message };
    }
}

async function runTests() {
  const testCases = [
    { newName: "NewAlice", expectedStatus: 200 },
    { newName: "Al", expectedStatus: 400 },
    { newName: "AliceWithVeryLongName", expectedStatus: 400 },
    { newName: "Alice@123", expectedStatus: 400 },
    { newName: "Bob", expectedStatus: 400 }, // Assuming 'Bob' already exists
    { newName: "Alice", expectedStatus: 200 },
  ]

  for (const testCase of testCases) {
    console.log(`\nTest case: ${testCase.newName}`)
    const result = await testUpdateUsername(testCase.newName)
    if (result.status === testCase.expectedStatus) {
      console.log("Test PASSED")
    } else {
      console.log("Test FAILED")
      console.log(`Expected status: ${testCase.expectedStatus}, Got: ${result.status}`)
    }
  }
}

runTests().catch(console.error)

