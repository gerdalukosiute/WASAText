import api from "@/services/axios.js"

// Create blob URLs cache to avoid duplicate requests
const blobUrlCache = new Map()

/**
 * Fetch media with proper authorization headers
 * @param {string} mediaId - The ID of the media to fetch
 * @returns {Promise<string>} - A blob URL for the media
 */
export const fetchMedia = async (mediaId) => {
  if (!mediaId) {
    return null
  }

  // Check if we already have this media in cache
  if (blobUrlCache.has(mediaId)) {
    return blobUrlCache.get(mediaId)
  }

  try {
    const userId = localStorage.getItem("userId")
    if (!userId) {
      throw new Error("User not authenticated")
    }

    // Use the existing axios instance which handles headers
    const response = await api.get(`/media/${mediaId}`, {
      responseType: "blob",
      headers: {
        "X-User-ID": userId,
      },
    })

    // Create a blob URL from the response
    const blobUrl = URL.createObjectURL(response.data)

    // Cache the blob URL
    blobUrlCache.set(mediaId, blobUrl)

    return blobUrl
  } catch (error) {
    console.error(`Error fetching media ${mediaId}:`, error)
    return null
  }
}

/**
 * Clean up blob URLs to prevent memory leaks
 * @param {string} mediaId - The ID of the media to clean up
 */
export const cleanupMedia = (mediaId) => {
  if (blobUrlCache.has(mediaId)) {
    URL.revokeObjectURL(blobUrlCache.get(mediaId))
    blobUrlCache.delete(mediaId)
  }
}

/**
 * Clean up all blob URLs
 */
export const cleanupAllMedia = () => {
  blobUrlCache.forEach((url) => {
    URL.revokeObjectURL(url)
  })
  blobUrlCache.clear()
}