const fetch = require("node-fetch")

const {
  API_URL,
  ALICE_ID,
} = require('./details');

async function testGetUsers(query, expectedStatus, expectedUserCount) {
  const endpoint = query ? `/users?q=${encodeURIComponent(query)}` : "/users"
  console.log(`Testing GET ${endpoint}`)

  try {
    const response = await fetch(`${API_URL}${endpoint}`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": ALICE_ID,
      },
    })

    const data = await response.json()
    console.log("Status:", response.status)
    console.log("Response:", data)

    if (response.status === expectedStatus) {
      if (expectedUserCount !== undefined && data.users && data.users.length === expectedUserCount) {
        console.log("Test PASSED")
      } else if (expectedUserCount === undefined) {
        console.log("Test PASSED")
      } else {
        console.log("Test FAILED")
        console.log(`Expected ${expectedUserCount} users, got ${data.users ? data.users.length : 0}`)
      }
    } else {
      console.log("Test FAILED")
      console.log(`Expected status: ${expectedStatus}, Got: ${response.status}`)
    }
  } catch (error) {
    console.error("Error:", error.message)
    console.log("Test FAILED")
  }
  console.log("---")
}

async function runTests() {
  // Test 1: Get all users
  await testGetUsers("", 200)

  // Test 2: Search for 'Alice'
  await testGetUsers("Alice", 200, 1)

  // Test 3: Search for 'Bob'
  await testGetUsers("Bob", 200, 1)

  // Test 4: Search for non-existent user
  await testGetUsers("NonExistentUser", 200, 0)

  // Test 5: Search with partial name
  await testGetUsers("Al", 200)

  // Test 6: Search with empty query
  await testGetUsers(" ", 200)

  // Test 7: Search with special characters
  await testGetUsers("Alice@123", 200, 0)

  // Test 8: Test pagination (if implemented)
  // You might need to adjust this based on your API's pagination parameters
  //await testGetUsers("?limit=5&offset=0", 200, 5)
}

runTests().catch(console.error)

