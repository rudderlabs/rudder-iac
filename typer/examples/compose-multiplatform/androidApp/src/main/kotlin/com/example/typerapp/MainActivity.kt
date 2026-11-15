package com.example.typerapp

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Placeholders. Point them at a data plane you control to see the events arrive.
        val transport = AndroidTransport(application, writeKey = "YOUR_WRITE_KEY", dataPlaneUrl = "http://10.0.2.2:19187", controlPlaneUrl = "http://10.0.2.2:19187")
        setContent { App(transport) }
    }
}
