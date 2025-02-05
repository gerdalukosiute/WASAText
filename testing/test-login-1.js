const fetch = require("node-fetch")

const {
  API_URL,
} = require('./details');

async function testLogin(username) {
  console.log(`Testing login for user: ${username}`)
  try {
    const response = await fetch(`${API_URL}/session`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: username }),
    })
    const data = await response.json()
    console.log("Status:", response.status)
    console.log("Response:", data)
    return { status: response.status, data }
  } catch (error) {
    console.error("Error:", error.message)
    return { status: 500, error: error.message }
  }
}

async function runTests() {
  const testCases = [
    { username: "Alice", expectedStatus: 201 },
    { username: "Bob", expectedStatus: 201 },
    { username: "Charlie", expectedStatus: 201 },
    { username: "", expectedStatus: 400 },
    { username: "a".repeat(20), expectedStatus: 400 },
  ]

  for (const testCase of testCases) {
    console.log(`\nTest case: ${testCase.username}`)
    const result = await testLogin(testCase.username)
    if (result.status === testCase.expectedStatus) {
      console.log("Test PASSED")
    } else {
      console.log("Test FAILED")
      console.log(`Expected status: ${testCase.expectedStatus}, Got: ${result.status}`)
    }
  }
}

runTests().catch(console.error)

