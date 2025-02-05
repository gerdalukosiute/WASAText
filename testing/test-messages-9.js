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

async function testSendMessage(userId, conversationId, messageType, content, description) {
  console.log(`Testing POST /conversations/${conversationId}/messages for ${description}`)
  const body = { messageType, content }
  const result = await makeRequest("POST", `/conversations/${conversationId}/messages`, body, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  if (result.status === 201 && result.data) {
    const hasRequiredFields =
      result.data.messageId &&
      result.data.conversationId &&
      result.data.sender &&
      result.data.content &&
      result.data.messageType &&
      result.data.timestamp

    if (!hasRequiredFields) {
      console.log("Test FAILED: Missing required fields in response")
    } else {
      console.log("Test PASSED: Response structure is valid")
    }
  } else {
    console.log(
      `Test ${[400, 401, 403, 404].includes(result.status) ? "PASSED" : "FAILED"}: Expected status code received`,
    )
  }

  console.log("---")
  return result
}

async function testGetMessages(userId, conversationId, limit = null, before = null, description) {
  let endpoint = `/conversations/${conversationId}/messages`
  const queryParams = []
  if (limit) queryParams.push(`limit=${limit}`)
  if (before) queryParams.push(`before=${before}`)
  if (queryParams.length > 0) {
    endpoint += "?" + queryParams.join("&")
  }

  console.log(`Testing GET ${endpoint} for ${description}`)
  const result = await makeRequest("GET", endpoint, null, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  if (result.status === 200 && result.data && Array.isArray(result.data.messages)) {
    for (const message of result.data.messages) {
      const hasRequiredFields =
        message.id &&
        message.sender &&
        message.type &&
        message.content &&
        message.timestamp &&
        message.status !== undefined &&
        Array.isArray(message.comments)

      if (!hasRequiredFields) {
        console.log("Test FAILED: Missing required fields in message object")
        console.log("Message:", message)
        return result
      }

      for (const comment of message.comments) {
        const hasCommentFields = comment.id && comment.userId && comment.content && comment.timestamp

        if (!hasCommentFields) {
          console.log("Test FAILED: Missing required fields in comment object")
          console.log("Comment:", comment)
          return result
        }
      }
    }
    console.log("Test PASSED: Response structure is valid")
  } else {
    console.log(`Test ${[401, 403, 404].includes(result.status) ? "PASSED" : "FAILED"}: Expected status code received`)
  }

  console.log("---")
  return result
}

async function runTests() {
  try {
    // Create a conversation for testing
    const createConversationResult = await makeRequest(
      "POST",
      "/conversations",
      {
        title: "Test Conversation",
        isGroup: false,
        participants: [BOB_ID],
      },
      ALICE_ID,
    )

    if (createConversationResult.status !== 201 || !createConversationResult.data.conversationId) {
      console.log("Failed to create test conversation. Aborting tests.")
      return
    }

    const conversationId = createConversationResult.data.conversationId

    // Test 1: Send a text message
    const sendTextResult = await testSendMessage(
      ALICE_ID,
      conversationId,
      "text",
      "Hello, Bob!",
      "sending text message",
    )

    // Test 2: Send a photo message
    await testSendMessage(BOB_ID, conversationId, "photo", "https://example.com/photo.jpg", "sending photo message")

    // Test 3: Try to send a message with invalid type
    await testSendMessage(ALICE_ID, conversationId, "invalid", "This should fail", "sending message with invalid type")

    // Test 4: Try to send a message with empty content
    await testSendMessage(ALICE_ID, conversationId, "text", "", "sending message with empty content")

    // Test 5: Get messages (default limit)
    await testGetMessages(ALICE_ID, conversationId, null, null, "getting messages with default limit")

    // Test 6: Get messages with custom limit
    await testGetMessages(BOB_ID, conversationId, 1, null, "getting messages with custom limit")

    // Test 7: Get messages before a specific time
    if (sendTextResult.status === 201 && sendTextResult.data.timestamp) {
      const beforeTime = new Date(sendTextResult.data.timestamp)
      beforeTime.setSeconds(beforeTime.getSeconds() - 1)
      await testGetMessages(
        ALICE_ID,
        conversationId,
        null,
        beforeTime.toISOString(),
        "getting messages before specific time",
      )
    }

    // Test 8: Try to get messages with invalid conversation ID
    await testGetMessages(
      ALICE_ID,
      "invalid-conversation-id",
      null,
      null,
      "getting messages with invalid conversation ID",
    )

    // Test 9: Try to get messages with invalid user ID
    await testGetMessages(INVALID_USER_ID, conversationId, null, null, "getting messages with invalid user ID")

    // Test 10: Send and retrieve a message with special characters
    await testSendMessage(
      ALICE_ID,
      conversationId,
      "text",
      "Hello! 你好! こんにちは! 🌟💬",
      "sending message with special characters",
    )
    await testGetMessages(BOB_ID, conversationId, null, null, "retrieving message with special characters")
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
}

// Run the tests
runTests().catch(console.error)

