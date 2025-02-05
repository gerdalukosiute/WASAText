const fetch = require("node-fetch")

const {
  API_URL,
  ALICE_ID,
  BOB_ID,
} = require('./details');

async function testUpdateUserPhoto(userId, photoUrl, expectedStatus) {
  console.log(`Testing update user photo for user ${userId} with URL: ${photoUrl}`)
  try {
    const response = await fetch(`${API_URL}/user/${userId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": userId,
      },
      body: JSON.stringify({ photoUrl }),
    })

    const data = await response.json()
    console.log("Status:", response.status)
    console.log("Response:", data)

    if (response.status === expectedStatus) {
      console.log("Test PASSED")
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
  // Test 1: Successful update
  await testUpdateUserPhoto(ALICE_ID, "https://example.com/alice.jpg", 200)

  // Test 2: Invalid URL
  await testUpdateUserPhoto(ALICE_ID, "not-a-valid-url", 400)

  // Test 3: Empty URL
  await testUpdateUserPhoto(ALICE_ID, "", 400)

  // Test 4: User trying to update another user's photo
  await testUpdateUserPhoto(BOB_ID, "https://example.com/bob-new.jpg", 200)

  // Test 5: Non-existent user
  await testUpdateUserPhoto("non-existent-user-id", "https://example.com/photo.jpg", 404)

  // Test 6: Update with the same photo URL
  await testUpdateUserPhoto(ALICE_ID, "https://example.com/alice.jpg", 200)
}

runTests().catch(console.error)

