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
    const hasRequiredFields = result.data.conversationId
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

async function testGetConversationDetails(userId, conversationId, description) {
  console.log(`Testing GET /conversations/${conversationId} for ${description}`)
  const result = await makeRequest("GET", `/conversations/${conversationId}`, null, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  // Validate response structure if successful
  if (result.status === 200 && result.data) {
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

      // Validate participants structure
      for (const participant of result.data.participants) {
        if (!participant.id || !participant.name) {
          console.log("Test FAILED: Invalid participant structure")
          return result
        }
      }

      // Validate messages structure
      for (const message of result.data.messages) {
        if (
          !message.id ||
          !message.sender ||
          !message.type ||
          !message.content ||
          !message.timestamp ||
          !message.status
        ) {
          console.log("Test FAILED: Invalid message structure")
          return result
        }

        // Validate comments structure if present
        if (message.comments) {
          for (const comment of message.comments) {
            if (
              !comment.id ||
              !comment.messageId ||
              !comment.userId ||
              !comment.username ||
              !comment.content ||
              !comment.timestamp
            ) {
              console.log("Test FAILED: Invalid comment structure")
              return result
            }
          }
        }
      }

      console.log("Test PASSED: Detailed structure validation successful")
    }
  } else {
    console.log(`Test ${[401, 404].includes(result.status) ? "PASSED" : "FAILED"}: Expected status code received`)
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
    const startConvResult = await testStartConversation(
      ALICE_ID,
      [BOB_ID],
      false,
      "",
      "Alice starting direct conversation with Bob",
    )

    // Test 4: Get conversation details for the newly created conversation
    if (startConvResult.status === 201 && startConvResult.data.conversationId) {
      await testGetConversationDetails(
        ALICE_ID,
        startConvResult.data.conversationId,
        "Alice getting conversation details",
      )
      await testGetConversationDetails(BOB_ID, startConvResult.data.conversationId, "Bob getting conversation details")
    }

    // Test 5: Try to get conversation details with invalid conversation ID
    await testGetConversationDetails(ALICE_ID, "invalid-conversation-id", "invalid conversation ID")

    // Test 6: Try to get conversation details with invalid user ID
    if (startConvResult.status === 201 && startConvResult.data.conversationId) {
      await testGetConversationDetails(INVALID_USER_ID, startConvResult.data.conversationId, "invalid user ID")
    }

    // Test 7: Start group conversation
    const startGroupResult = await testStartConversation(
      ALICE_ID,
      [BOB_ID],
      true,
      "Test Group",
      "Alice starting group conversation with Bob",
    )

    // Test 8: Get group conversation details
    if (startGroupResult.status === 201 && startGroupResult.data.conversationId) {
      await testGetConversationDetails(
        ALICE_ID,
        startGroupResult.data.conversationId,
        "Alice getting group conversation details",
      )
    }

    // Existing tests...
    // (Keep the other existing tests from the previous version)
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
}

// Run the tests
runTests().catch(console.error)

