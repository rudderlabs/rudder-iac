package com.example.typerapp

import android.app.Application
import com.example.typerapp.typer.RudderTyperTransport
import com.rudderstack.sdk.kotlin.android.Analytics
import com.rudderstack.sdk.kotlin.android.Configuration
import com.rudderstack.sdk.kotlin.core.internals.logger.Logger
import com.rudderstack.sdk.kotlin.core.internals.models.RudderOption
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive

/** Implements the generated transport on top of the RudderStack Kotlin SDK. */
class AndroidTransport(
    application: Application,
    writeKey: String,
    dataPlaneUrl: String,
    controlPlaneUrl: String = "https://api.rudderstack.com",
) : RudderTyperTransport {
    private val analytics = Analytics(
        Configuration(
            writeKey = writeKey,
            application = application,
            dataPlaneUrl = dataPlaneUrl,
            controlPlaneUrl = controlPlaneUrl,
            logLevel = Logger.LogLevel.DEBUG,
        )
    )

    override fun track(name: String, properties: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.track(name, properties.toJsonObject(), context.toOption())
    }

    override fun identify(userId: String, traits: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.identify(userId, traits.toJsonObject(), context.toOption())
    }

    override fun group(groupId: String, traits: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.group(groupId, traits.toJsonObject(), context.toOption())
    }

    override fun screen(screenName: String, category: String, properties: Map<String, Any?>, context: Map<String, Any?>) {
        analytics.screen(screenName, category, properties.toJsonObject(), context.toOption())
    }
}

private fun Map<String, Any?>.toOption() = RudderOption(customContext = toJsonObject())

private fun Map<String, Any?>.toJsonObject() = JsonObject(mapValues { it.value.toJsonElement() })

private fun Any?.toJsonElement(): JsonElement = when (this) {
    null -> JsonNull
    is JsonElement -> this
    is Boolean -> JsonPrimitive(this)
    is Number -> JsonPrimitive(this)
    is String -> JsonPrimitive(this)
    is Map<*, *> -> JsonObject(entries.associate { (k, v) -> k.toString() to v.toJsonElement() })
    is Iterable<*> -> JsonArray(map { it.toJsonElement() })
    else -> JsonPrimitive(toString())
}
