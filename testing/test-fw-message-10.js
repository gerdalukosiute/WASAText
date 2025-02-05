const fetch = require("node-fetch")
const { API_URL, ALICE_ID, BOB_ID, CHARLIE_ID, INVALID_USER_ID } = require("./details")

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

async function login(username) {
  const result = await makeRequest("POST", "/session", { name: username })
  if (result.status !== 201) {
    throw new Error(`Failed to login user ${username}`)
  }
  return result.data.identifier
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
  return result.data.conversationId
}

async function sendMessage(userId, conversationId, content) {
  const result = await makeRequest(
    "POST",
    `/conversations/${conversationId}/messages`,
    { messageType: "text", content },
    userId,
  )
  return result.data.messageId
}

async function testForwardMessage(userId, messageId, targetConversationId, description) {
  console.log(`Testing POST /messages/${messageId}/forward for ${description}`)
  const result = await makeRequest(
    "POST",
    `/messages/${messageId}/forward`,
    {
      originalMessageId: messageId,
      targetConversationId: targetConversationId,
    },
    userId,
  )
  console.log("Status:", result.status)
  console.log("Response:", JSON.stringify(result.data, null, 2))

  if (result.status === 201 && result.data) {
    const hasRequiredFields =
      result.data.newMessageId &&
      result.data.originalMessageId &&
      result.data.targetConversationId &&
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

async function runTests() {
  try {
    // Ensure all users are logged in (this will create them if they don't exist)
    await login("Alice")
    await login("Bob")
    await login("Charlie")

    // Create three conversations
    const conversation1Id = await createConversation(ALICE_ID, BOB_ID)
    const conversation2Id = await createConversation(BOB_ID, CHARLIE_ID)
    const conversation3Id = await createConversation(CHARLIE_ID, ALICE_ID)

    // Send a message in the first conversation
    const messageId = await sendMessage(ALICE_ID, conversation1Id, "Hello, this is a test message!")

    // Test 1: Forward message (happy path)
    await testForwardMessage(ALICE_ID, messageId, conversation3Id, "forwarding a valid message")

    // Test 2: Try to forward with invalid message ID
    await testForwardMessage(ALICE_ID, "invalid-message-id", conversation3Id, "forwarding with invalid message ID")

    // Test 3: Try to forward to invalid conversation ID
    await testForwardMessage(ALICE_ID, messageId, "invalid-conversation-id", "forwarding to invalid conversation ID")

    // Test 4: Try to forward as unauthorized user
    await testForwardMessage(INVALID_USER_ID, messageId, conversation3Id, "forwarding as unauthorized user")

    // Test 5: Try to forward to a conversation where the user is not a participant
    const bobMessageId = await sendMessage(BOB_ID, conversation2Id, "Another message from Bob to Charlie")
    await testForwardMessage(ALICE_ID, bobMessageId, conversation1Id, "forwarding from unauthorized conversation")

    console.log("All tests completed.")
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
}

// Run the tests
runTests().catch(console.error)

