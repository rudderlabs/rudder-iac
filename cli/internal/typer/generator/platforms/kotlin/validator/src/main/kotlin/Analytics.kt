package com.rudderstack.sdk.kotlin.core

import com.rudderstack.sdk.kotlin.core.internals.models.RudderOption
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

class Analytics() {
    fun identify(userId: String, traits: JsonObject, options: RudderOption) {
        val output = buildJsonObject {
            put("type", "identify")
            put("userId", userId)
            put("traits", traits)
            put("context", options.customContext)
        }
        println(output)
    }

    fun track(name: String, properties: JsonObject, options: RudderOption) {
        val output = buildJsonObject {
            put("type", "track")
            put("name", name)
            put("properties", properties)
            put("context", options.customContext)
        }
        println(output)
    }

    fun group(groupId: String, traits: JsonObject, options: RudderOption) {
        val output = buildJsonObject {
            put("type", "group")
            put("groupId", groupId)
            put("traits", traits)
            put("context", options.customContext)
        }
        println(output)
    }

    fun screen(screenName: String, category: String, properties: JsonObject, options: RudderOption) {
        val output = buildJsonObject {
            put("type", "screen")
            put("screenName", screenName)
            put("category", category)
            put("properties", properties)
            put("context", options.customContext)
        }
        println(output)
    }
}