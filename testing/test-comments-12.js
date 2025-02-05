const fetch = require('node-fetch');

const {
    API_URL,
    ALICE_ID,
    BOB_ID,
    CHARLIE_ID,
  } = require('./details');

async function makeRequest(method, endpoint, body = null, userId) {
  const headers = {
    'Content-Type': 'application/json',
    'X-User-ID': userId,
  };

  const options = {
    method,
    headers,
    body: body ? JSON.stringify(body) : null,
  };

  const response = await fetch(`${API_URL}${endpoint}`, options);
  return {
    status: response.status,
    data: await response.json().catch(() => null),
  };
}

async function createConversation(creatorId, participantId) {
  const result = await makeRequest(
    'POST',
    '/conversations',
    {
      title: 'Test Conversation',
      isGroup: false,
      participants: [participantId],
    },
    creatorId
  );
  if (result.status !== 201 || !result.data.conversationId) {
    throw new Error(`Failed to create conversation: ${JSON.stringify(result)}`);
  }
  return result.data.conversationId;
}

async function sendMessage(userId, conversationId, content) {
  const result = await makeRequest(
    'POST',
    `/conversations/${conversationId}/messages`,
    { messageType: 'text', content },
    userId
  );
  if (result.status !== 201 || !result.data.messageId) {
    throw new Error(`Failed to send message: ${JSON.stringify(result)}`);
  }
  return result.data.messageId;
}

async function addComment(userId, messageId, comment, expectedStatus, description) {
  console.log(`Testing POST /messages/${messageId}/comments for ${description}`);
  const result = await makeRequest(
    'POST',
    `/messages/${messageId}/comments`,
    { comment },
    userId
  );
  console.log('Status:', result.status);
  console.log('Response:', JSON.stringify(result.data, null, 2));

  if (result.status === expectedStatus) {
    if (result.status === 201 && result.data) {
      const hasRequiredFields = 
        result.data.commentId &&
        result.data.messageId &&
        result.data.userId &&
        result.data.content &&
        result.data.timestamp;
      if (!hasRequiredFields) {
        console.log('Test FAILED: Missing required fields in response');
      } else {
        console.log('Test PASSED: Response structure is valid');
      }
    } else {
      console.log('Test PASSED: Expected status code received');
    }
  } else {
    console.log(`Test FAILED: Expected status ${expectedStatus}, but got ${result.status}`);
  }

  console.log('---');
  return result;
}

async function deleteComment(userId, messageId, commentId, expectedStatus, description) {
  console.log(`Testing DELETE /messages/${messageId}/comments/${commentId} for ${description}`);
  const result = await makeRequest(
    'DELETE',
    `/messages/${messageId}/comments/${commentId}`,
    null,
    userId
  );
  console.log('Status:', result.status);
  console.log('Response:', JSON.stringify(result.data, null, 2));

  if (result.status === expectedStatus) {
    if (result.status === 200 && result.data) {
      const hasRequiredFields = 
        result.data.messageId &&
        result.data.commentId &&
        result.data.username;
      if (!hasRequiredFields) {
        console.log('Test FAILED: Missing required fields in response');
      } else {
        console.log('Test PASSED: Response structure is valid');
      }
    } else {
      console.log('Test PASSED: Expected status code received');
    }
  } else {
    console.log(`Test FAILED: Expected status ${expectedStatus}, but got ${result.status}`);
  }

  console.log('---');
  return result;
}

async function runTests() {
  try {
    // Create a conversation between Alice and Bob
    const conversationId = await createConversation(ALICE_ID, BOB_ID);

    // Alice sends a message
    const messageId = await sendMessage(ALICE_ID, conversationId, "Hello, Bob!");

    // Test 1: Alice adds a comment to her own message (should succeed)
    const aliceCommentResult = await addComment(ALICE_ID, messageId, "This is a test comment", 201, "Alice adding comment to her message");
    const aliceCommentId = aliceCommentResult.data.commentId;

    // Test 2: Bob adds a comment to Alice's message (should succeed)
    const bobCommentResult = await addComment(BOB_ID, messageId, "Reply from Bob", 201, "Bob adding comment to Alice's message");
    const bobCommentId = bobCommentResult.data.commentId;

    // Test 3: Charlie tries to add a comment (should fail with 401 Unauthorized)
    await addComment(CHARLIE_ID, messageId, "Unauthorized comment", 401, "Charlie trying to add comment");

    // Test 4: Try to add an empty comment (should fail with 400 Bad Request)
    await addComment(ALICE_ID, messageId, "", 400, "Adding empty comment");

    // Test 5: Try to add a comment to a non-existent message
    await addComment(ALICE_ID, "non-existent-message-id", "This should fail", 404, "Adding comment to non-existent message");

    // Test 6: Alice deletes her own comment (should succeed)
    await deleteComment(ALICE_ID, messageId, aliceCommentId, 200, "Alice deleting her own comment");

    // Test 7: Alice tries to delete Bob's comment (should fail with 401 Unauthorized)
    await deleteComment(ALICE_ID, messageId, bobCommentId, 401, "Alice trying to delete Bob's comment");

    // Test 8: Bob deletes his own comment (should succeed)
    await deleteComment(BOB_ID, messageId, bobCommentId, 200, "Bob deleting his own comment");

    // Test 9: Try to delete a non-existent comment
    await deleteComment(ALICE_ID, messageId, "non-existent-comment-id", 404, "Deleting non-existent comment");

    // Test 10: Charlie tries to delete a comment (should fail with 401 Unauthorized)
    await deleteComment(CHARLIE_ID, messageId, aliceCommentId, 401, "Charlie trying to delete a comment");

    console.log("All tests completed.");
  } catch (error) {
    console.error("An error occurred during testing:", error);
  }
}

// Run the tests
runTests().catch(console.error);