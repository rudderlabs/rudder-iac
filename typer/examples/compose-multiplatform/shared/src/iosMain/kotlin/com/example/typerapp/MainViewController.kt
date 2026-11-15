package com.example.typerapp

import androidx.compose.ui.window.ComposeUIViewController
import com.example.typerapp.typer.RudderTyperTransport

// Swift passes in its own transport, built on the Swift SDK.
fun MainViewController(transport: RudderTyperTransport) = ComposeUIViewController { App(transport) }
