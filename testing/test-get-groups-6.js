const fetch = require("node-fetch")

const {
  API_URL,
  ALICE_ID,
  BOB_ID,
  INVALID_USER_ID
} = require('./details');

async function makeRequest(endpoint, userId) {
  const response = await fetch(`${API_URL}${endpoint}`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      "X-User-ID": userId,
    },
  })
  return {
    status: response.status,
    data: await response.json().catch(() => null),
  }
}

async function testGetGroups(userId, expectedStatus, description) {
  console.log(`Testing GET /groups for ${description}`)
  const result = await makeRequest("/groups", userId)
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === expectedStatus ? "Test PASSED" : "Test FAILED")
  console.log("---")
  return result
}

async function runTests() {
  try {
    // Test with valid user (Alice)
    const aliceResult = await testGetGroups(ALICE_ID, 200, "valid user (Alice)")
    if (aliceResult.status === 200) {
      console.log("Alice's groups:", aliceResult.data)
    }

    // Test with another valid user (Bob)
    const bobResult = await testGetGroups(BOB_ID, 200, "valid user (Bob)")
    if (bobResult.status === 200) {
      console.log("Bob's groups:", bobResult.data)
    }

    // Test with invalid user ID
    await testGetGroups(INVALID_USER_ID, 404, "invalid user ID")

    // Test without user ID (should result in an error)
    await testGetGroups(null, 404, "missing user ID")
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
  console.log("All tests completed.")
}

runTests().catch(console.error)

