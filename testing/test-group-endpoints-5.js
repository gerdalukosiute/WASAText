const fetch = require("node-fetch")

const {
  API_URL,
  ALICE_ID,
  BOB_ID,
  ALICE_USERNAME,
  BOB_USERNAME,
  TEST_GROUP_ID,
} = require('./details');

async function makeRequest(method, endpoint, body = null, userId = ALICE_ID) {
  const options = {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-User-ID": userId,
    },
  }
  if (body) {
    options.body = JSON.stringify(body)
  }
  const response = await fetch(`${API_URL}${endpoint}`, options)
  return {
    status: response.status,
    data: await response.json().catch(() => null),
  }
}
// doesnt account for pass if 409 and the group exists before
async function testCreateGroupAndAddUser() {
  console.log("Testing POST /groups/{groupId} (Create Group and Add User)")
  const result = await makeRequest("POST", `/groups/${TEST_GROUP_ID}`, { username: ALICE_USERNAME }, ALICE_ID)
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === 200 ? "Test PASSED" : "Test FAILED")
  console.log("---")
}

async function testAddUserToExistingGroup() {
  console.log(`Testing POST /groups/${TEST_GROUP_ID} (Add User to Existing Group)`)
  const result = await makeRequest("POST", `/groups/${TEST_GROUP_ID}`, { username: BOB_USERNAME }, ALICE_ID)
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === 200 || result.status === 409 ? "Test PASSED" : "Test FAILED")
  console.log("---")
}

async function testSetGroupName() {
  console.log(`Testing PUT /groups/${TEST_GROUP_ID} (Set Group Name)`)
  const result = await makeRequest("PUT", `/groups/${TEST_GROUP_ID}`, { name: "Updated Test Group" }, ALICE_ID)
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === 200 ? "Test PASSED" : "Test FAILED")
  console.log("---")
}

async function testSetGroupPhoto() {
  console.log(`Testing PATCH /groups/${TEST_GROUP_ID} (Set Group Photo)`)
  const result = await makeRequest(
    "PATCH",
    `/groups/${TEST_GROUP_ID}`,
    { photo: "https://example.com/group-photo.jpg" },
    ALICE_ID,
  )
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === 200 ? "Test PASSED" : "Test FAILED")
  console.log("---")
}

async function testLeaveGroup() {
  console.log(`Testing DELETE /groups/${TEST_GROUP_ID} (Leave Group)`)
  const result = await makeRequest("DELETE", `/groups/${TEST_GROUP_ID}`, null, BOB_ID)
  console.log("Status:", result.status)
  console.log("Response:", result.data)
  console.log(result.status === 200 ? "Test PASSED" : "Test FAILED")
  console.log("---")
}

async function runTests() {
  try {
    await testCreateGroupAndAddUser()
    await testAddUserToExistingGroup()
    await testSetGroupName()
    await testSetGroupPhoto()
    await testLeaveGroup()
  } catch (error) {
    console.error("An error occurred during testing:", error)
  }
  console.log("All tests completed.")
}

runTests().catch(console.error)

