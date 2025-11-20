package com.rudderstack.ruddertyper

import com.rudderstack.sdk.kotlin.core.Analytics
import com.rudderstack.sdk.kotlin.core.internals.models.Event
import com.rudderstack.sdk.kotlin.core.internals.models.EventType
import com.rudderstack.sdk.kotlin.core.internals.models.*
import org.junit.jupiter.api.*
import com.rudderstack.sdk.kotlin.core.internals.plugins.Plugin
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject

private var emptyJsonObject = buildJsonObject {}

sealed class EventValidation(val type: EventType) {
    data class TrackValidation(
        val name: String,
        val properties: JsonObject = emptyJsonObject,
    ): EventValidation(type = EventType.Track)

    data class IdentifyValidation(
        val userId: String,
        val traits: JsonObject = emptyJsonObject
    ): EventValidation(type = EventType.Identify)

    data class ScreenValidation(
        val screenName: String,
        val properties: JsonObject = emptyJsonObject
    ): EventValidation(type = EventType.Screen)

    data class GroupValidation(
        val groupId: String,
        val traits: JsonObject = emptyJsonObject
    ): EventValidation(type = EventType.Group)
}

class EventValidationPlugin : Plugin {
    override val pluginType: Plugin.PluginType = Plugin.PluginType.OnProcess
    override lateinit var analytics: Analytics

    private var received: MutableList<Event> = mutableListOf()
    private var validationIndex: Int = 0

    override fun setup(analytics: Analytics) {
        super.setup(analytics)
    }

    override suspend fun intercept(event: Event): Event? {
        println("EventFilteringPlugin: Intercepting event: $event")
        received.add(event)
        return null
    }

    fun validateCount(count: Int, timeoutMs: Long = 5000) {
        val startTime = System.currentTimeMillis()
        val pollIntervalMs = 100L

        // Wait for events to arrive with timeout
        while (received.size < count) {
            if (System.currentTimeMillis() - startTime > timeoutMs) {
                throw AssertionError(
                    "Timeout waiting for events. " +
                    "Expected $count events, but only received ${received.size} after ${timeoutMs}ms"
                )
            }
            Thread.sleep(pollIntervalMs)
        }

        // Check if more events were received than expected
        if (received.size > count) {
            throw AssertionError(
                "Received more events than expected. " +
                "Expected $count events, but received ${received.size}. " +
                "Extra event: ${received[count]}"
            )
        }
    }

    public fun validateNext(expected: EventValidation) {
        validateEvent(received[validationIndex], expected)
        validationIndex++
    }

    private fun validateEvent(received: Event, expected: EventValidation) {
        when (received) {
            is TrackEvent -> {
                if (expected !is EventValidation.TrackValidation) {
                    throw AssertionError("Expected track event, but got ${expected.type}")
                }
                Assertions.assertEquals(expected.name, received.event, "Track event name mismatch")
                Assertions.assertEquals(expected.properties, received.properties, "Track event properties mismatch")
            }
            is IdentifyEvent -> {
                if (expected !is EventValidation.IdentifyValidation) {
                    throw AssertionError("Expected identify event, but got ${expected.type}")
                }
                Assertions.assertEquals(expected.userId, received.userId, "Identify userId mismatch")
                val traits = received.context["traits"]
                Assertions.assertEquals(expected.traits, traits, "Identify traits mismatch")
            }
            is ScreenEvent -> {
                if (expected !is EventValidation.ScreenValidation) {
                    throw AssertionError("Expected screen event, but got ${expected.type}")
                }
                Assertions.assertEquals(expected.screenName, received.screenName, "Screen event name mismatch")
                Assertions.assertEquals(expected.properties, received.properties, "Screen event properties mismatch")
            }
            is GroupEvent -> {
                if (expected !is EventValidation.GroupValidation) {
                    throw AssertionError("Expected group event, but got ${expected.type}")
                }
                Assertions.assertEquals(expected.groupId, received.groupId, "Group event groupId mismatch")
                Assertions.assertEquals(expected.traits, received.traits, "Group event traits mismatch")
            }
            else -> throw AssertionError("Unexpected event type: ${received.type}")
        }
    }
}
