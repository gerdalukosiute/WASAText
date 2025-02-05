const fetch = require("node-fetch")
const { API_URL, ALICE_ID, BOB_ID, INVALID_USER_ID } = require("./details")

async function makeRequest(method, endpoint, body = null, userId) {
  const headers = {
    "Content-Type": "application/json",
    "X-User-ID": userId,
  }

  const options = {
    method,
    headers,
    body: body ? JSON.stringify(body) : null,
  }

  const response = await fetch(`${API_URL}${endpoint}`, options)
  return {
    status: response.status,
    data: await response.json().catch(() => null),
  }
}

async function testGetConversations(userId, description) {
  console.log(`Testing GET /conversations for ${description}`)
  const result = await makeRequest("GET", "/conversations", null, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  // Validate response structure if successful
  if (result.status === 200 && Array.isArray(result.data)) {
    for (const conv of result.data) {
      const hasRequiredFields =
        conv.conversationId &&
        "title" in conv &&
        "profilePhoto" in conv &&
        "isGroup" in conv &&
        conv.lastMessage &&
        conv.updatedAt

      if (!hasRequiredFields) {
        console.log("Test FAILED: Missing required fields in conversation object")
        console.log("Conversation:", conv)
        return result
      }
    }
    console.log("Test PASSED: Response structure is valid")
  } else if (result.status !== 200) {
    console.log(`Test ${result.status === 401 ? "PASSED" : "FAILED"}: Expected unauthorized access to fail`)
  }

  console.log("---")
  return result
}

async function testStartConversation(userId, participants, isGroup = false, title = "", description) {
  console.log(`Testing POST /conversations for ${description}`)
  const body = {
    title,
    isGroup,
    participants,
  }

  const result = await makeRequest("POST", "/conversations", body, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  // Validate response structure if successful
  if (result.status === 201 && result.data) {
    const hasRequiredFields =
      result.data.conversationId &&
      "title" in result.data &&
      "isGroup" in result.data &&
      Array.isArray(result.data.participants) &&
      Array.isArray(result.data.messages)

    if (!hasRequiredFields) {
      console.log("Test FAILED: Missing required fields in response")
    } else {
      console.log("Test PASSED: Response structure is valid")
    }
  } else {
    console.log(`Test ${[400, 401, 404].includes(result.status) ? "PASSED" : "FAILED"}: Expected status code received`)
  }

  console.log("---")
  return result
}

async function runTests() {
  try {
    // Test 1: Get conversations for Alice (should succeed)
    await testGetConversations(ALICE_ID, "Alice (valid user)")

    // Test 2: Get conversations for Bob (should succeed)
    await testGetConversations(BOB_ID, "Bob (valid user)")

    // Test 3: Start direct conversation between Alice and Bob
    await testStartConversation(ALICE_ID, [BOB_ID], false, "", "Alice starting direct conversation with Bob")

    // Test 4: Start group conversation
    await testStartConversation(ALICE_ID, [BOB_ID], true, "Test Group", "Alice starting group conversation with Bob")

    // Test 5: Try to start direct conversation with multiple participants (should fail)
    await testStartConversation(
      ALICE_ID,
      [BOB_ID, "another-user-id"],
      false,
      "",
      "direct conversation with multiple participants",
    )

    // Test 6: Try to start group conversation with no participants (should fail)
    await testStartConversation(ALICE_ID, [], true, "Empty Group", "group conversation with no participants")

    // Test 7: Test with invalid user ID
    await testGetConversations(INVALID_USER_ID, "invalid user")

    // Test 8: Test starting conversation with non-existent user
    await testStartConversation(ALICE_ID, ["non-existent-user-id"], false, "", "conversation with non-existent user")

    // Test 9: Test starting conversation with self (should be handled by server)
    await testStartConversation(ALICE_ID, [ALICE_ID], false, "", "conversation with self")

    // Test 10: Start group conversation with multiple participants
    await testStartConversation(
      ALICE_ID,
      [BOB_ID], //could contain a 3rd user; for now not logged in
      true,
      "Multiple Users Group",
      "group conversation with multiple participants",
    )
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
}

// Run the tests
runTests().catch(console.error)

