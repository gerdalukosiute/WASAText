const fetch = require("node-fetch")

const { API_URL, CHARLIE_ID, ALICE_ID, BOB_ID, INVALID_USER_ID } = require("./details")

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

async function createConversation(creatorId, participantId) {
  const result = await makeRequest(
    "POST",
    "/conversations",
    {
      title: "Test Conversation",
      isGroup: false,
      participants: [participantId],
    },
    creatorId,
  )
  if (result.status !== 201 || !result.data.conversationId) {
    throw new Error(`Failed to create conversation: ${JSON.stringify(result)}`)
  }
  return result.data.conversationId
}

async function sendMessage(userId, conversationId, content) {
  const result = await makeRequest(
    "POST",
    `/conversations/${conversationId}/messages`,
    { messageType: "text", content },
    userId,
  )
  if (result.status !== 201 || !result.data.messageId) {
    throw new Error(`Failed to send message: ${JSON.stringify(result)}`)
  }
  return result.data.messageId
}

async function deleteMessage(userId, messageId, expectedStatus, description) {
  console.log(`Testing DELETE /messages/${messageId} for ${description}`)
  const result = await makeRequest("DELETE", `/messages/${messageId}`, null, userId)
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  if (result.status === expectedStatus) {
    if (result.status === 200 && result.data) {
      const hasRequiredFields = result.data.messageId && result.data.username
      if (!hasRequiredFields) {
        console.log("Test FAILED: Missing required fields in response")
      } else {
        console.log("Test PASSED: Response structure is valid")
      }
    } else {
      console.log("Test PASSED: Expected status code received")
    }
  } else {
    console.log(`Test FAILED: Expected status ${expectedStatus}, but got ${result.status}`)
  }

  console.log("---")
  return result
}

async function runTests() {
  try {
    // Create a conversation between Alice and Bob
    const conversationId = await createConversation(ALICE_ID, BOB_ID)

    // Alice sends two messages
    const aliceMessageId1 = await sendMessage(ALICE_ID, conversationId, "Hello, Bob!")
    const aliceMessageId2 = await sendMessage(ALICE_ID, conversationId, "How are you?")

    // Bob sends a message
    const bobMessageId = await sendMessage(BOB_ID, conversationId, "Hi, Alice!")

    // Test 1: Alice deletes her first message (should succeed)
    await deleteMessage(ALICE_ID, aliceMessageId1, 200, "Alice deleting her first message")

    // Test 2: Bob tries to delete Alice's second message (should fail with 403 Forbidden)
    await deleteMessage(BOB_ID, aliceMessageId2, 403, "Bob trying to delete Alice's message")

    // Test 3: Alice tries to delete a non-existent message
    await deleteMessage(ALICE_ID, "non-existent-message-id", 404, "Alice trying to delete a non-existent message")

    // Test 4: Charlie (not in the conversation) tries to delete Bob's message
    await deleteMessage(CHARLIE_ID, bobMessageId, 403, "Charlie trying to delete Bob's message")

    // Test 5: Invalid user tries to delete a message
    await deleteMessage(INVALID_USER_ID, bobMessageId, 403, "Invalid user trying to delete a message")

    // Test 6: Bob deletes his own message (should succeed)
    await deleteMessage(BOB_ID, bobMessageId, 200, "Bob deleting his own message")

    console.log("All tests completed.")
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
}

// Run the tests
runTests().catch(console.error)

