package com.deskdrop.app.api

import android.content.Context
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

object ApiClient {
    // For physical device, use your computer's IP:
    private const val BASE_URL = "http://192.168.0.166:3000/api/"
    // For emulator, use: "http://10.0.2.2:3000/api/"
    
    private var retrofit: Retrofit? = null
    
    fun getApiService(context: Context): ApiService {
        if (retrofit == null) {
            val loggingInterceptor = HttpLoggingInterceptor().apply {
                level = HttpLoggingInterceptor.Level.BODY
            }
            
            val client = OkHttpClient.Builder()
                .addInterceptor(loggingInterceptor)
                .addInterceptor { chain ->
                    val request = chain.request().newBuilder()
                        .addHeader("Authorization", "Bearer ${getDevToken(context)}")
                        .addHeader("Content-Type", "application/json")
                        .build()
                    chain.proceed(request)
                }
                .connectTimeout(30, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .build()
            
            retrofit = Retrofit.Builder()
                .baseUrl(BASE_URL)
                .client(client)
                .addConverterFactory(GsonConverterFactory.create())
                .build()
        }
        
        return retrofit!!.create(ApiService::class.java)
    }
    
    private fun getDevToken(context: Context): String {
        val prefs = context.getSharedPreferences("deskdrop_prefs", Context.MODE_PRIVATE)
        return prefs.getString("dev_token", "dev-token-for-testing-change-in-production") ?: "dev-token-for-testing-change-in-production"
    }
    
    fun setDevToken(context: Context, token: String) {
        val prefs = context.getSharedPreferences("deskdrop_prefs", Context.MODE_PRIVATE)
        prefs.edit().putString("dev_token", token).apply()
    }
}

