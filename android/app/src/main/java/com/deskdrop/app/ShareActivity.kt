package com.deskdrop.app

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.deskdrop.app.api.ApiService
import com.deskdrop.app.api.ApiClient
import retrofit2.Call
import retrofit2.Callback
import retrofit2.Response

class ShareActivity : AppCompatActivity() {
    
    private lateinit var apiService: ApiService
    private lateinit var urlTextView: TextView
    private lateinit var submitButton: Button
    private lateinit var statusTextView: TextView
    
    private var sharedUrl: String? = null
    private var deviceId: String? = null
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_share)
        
        // Initialize API service
        apiService = ApiClient.getApiService(this)
        
        // Get device ID from SharedPreferences or generate one
        deviceId = getOrCreateDeviceId()
        
        // Setup UI
        urlTextView = findViewById(R.id.urlTextView)
        submitButton = findViewById(R.id.submitButton)
        statusTextView = findViewById(R.id.statusTextView)
        
        // Handle incoming share intent
        handleShareIntent()
        
        // Setup submit button
        submitButton.setOnClickListener {
            sharedUrl?.let { url ->
                submitLink(url)
            } ?: run {
                showError("No URL to submit")
            }
        }
    }
    
    private fun handleShareIntent() {
        val intent = intent
        val action = intent.action
        val type = intent.type
        
        if (Intent.ACTION_SEND == action && type != null) {
            if ("text/plain" == type) {
                val sharedText = intent.getStringExtra(Intent.EXTRA_TEXT)
                if (sharedText != null) {
                    // Extract URL from shared text
                    val url = extractUrl(sharedText)
                    if (url != null) {
                        sharedUrl = url
                        urlTextView.text = url
                        statusTextView.text = "Ready to submit"
                    } else {
                        showError("No valid URL found in shared text")
                    }
                }
            } else if (type.startsWith("image/")) {
                val imageUri: Uri? = intent.getParcelableExtra(Intent.EXTRA_STREAM)
                // For MVP, we focus on text/URL sharing only
                showError("Image sharing not supported in MVP")
            }
        } else {
            showError("Invalid share intent")
        }
    }
    
    private fun extractUrl(text: String): String? {
        // Simple URL extraction - look for http:// or https://
        val urlPattern = Regex("https?://[^\\s]+")
        val match = urlPattern.find(text)
        return match?.value
    }
    
    private var retryCount = 0
    private val MAX_RETRIES = 2
    
    private fun submitLink(url: String) {
        if (deviceId == null) {
            showError("Device ID not available")
            return
        }
        
        // Update UI to show loading state
        submitButton.isEnabled = false
        if (retryCount == 0) {
            statusTextView.text = "Submitting..."
        } else {
            statusTextView.text = "Retrying... (${retryCount}/${MAX_RETRIES})"
        }
        
        val request = com.deskdrop.app.api.models.LinkRequest(
            url = url,
            deviceId = deviceId!!
        )
        
        apiService.submitLink(request).enqueue(object : Callback<com.deskdrop.app.api.models.LinkResponse> {
            override fun onResponse(
                call: Call<com.deskdrop.app.api.models.LinkResponse>,
                response: Response<com.deskdrop.app.api.models.LinkResponse>
            ) {
                submitButton.isEnabled = true
                retryCount = 0 // Reset on success
                
                if (response.isSuccessful) {
                    val linkResponse = response.body()
                    if (linkResponse != null && linkResponse.success) {
                        statusTextView.text = "Link submitted successfully!"
                        Toast.makeText(this@ShareActivity, "Link sent to desktop", Toast.LENGTH_SHORT).show()
                        
                        // Close activity after short delay
                        finish()
                    } else {
                        showError("Failed to submit link")
                    }
                } else {
                    // Server error - might be cold start, retry
                    if (response.code() == 503 || response.code() == 504) {
                        retryWithDelay(url)
                    } else {
                        val errorBody = response.errorBody()?.string()
                        showError("Server error: ${response.code()}")
                    }
                }
            }
            
            override fun onFailure(call: Call<com.deskdrop.app.api.models.LinkResponse>, t: Throwable) {
                // Network/timeout error - retry for cold starts
                if (retryCount < MAX_RETRIES && (t.message?.contains("timeout", ignoreCase = true) == true || 
                    t.message?.contains("failed to connect", ignoreCase = true) == true)) {
                    retryWithDelay(url)
                } else {
                    submitButton.isEnabled = true
                    retryCount = 0
                    showError("Network error: ${t.message}")
                }
            }
        })
    }
    
    private fun retryWithDelay(url: String) {
        retryCount++
        statusTextView.text = "Server waking up... Retrying (${retryCount}/${MAX_RETRIES})"
        
        // Exponential backoff: 2s, 4s
        val delayMs = (2000 * retryCount).toLong()
        
        android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
            submitLink(url)
        }, delayMs)
    }
    
    private fun showError(message: String) {
        statusTextView.text = "Error: $message"
        Toast.makeText(this, message, Toast.LENGTH_LONG).show()
    }
    
    private fun getOrCreateDeviceId(): String {
        val prefs = getSharedPreferences("deskdrop_prefs", MODE_PRIVATE)
        var deviceId = prefs.getString("device_id", null)
        
        if (deviceId == null) {
            // Generate a simple device ID
            deviceId = "android-${System.currentTimeMillis()}"
            prefs.edit().putString("device_id", deviceId).apply()
        }
        
        return deviceId
    }
}

