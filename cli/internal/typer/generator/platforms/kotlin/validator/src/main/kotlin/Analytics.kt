package com.rudderstack.sdk.kotlin.core

import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

class Analytics() {
    fun identify(userId: String, traits: JsonObject) {
        val output = buildJsonObject {
            put("type", "identify")
            put("userId", userId)
            put("traits", traits)
        }
        println(output)
    }

    fun track(name: String, properties: JsonObject) {
        val output = buildJsonObject {
            put("type", "track")
            put("name", name)
            put("properties", properties)
        }
        println(output)
    }

    fun group(groupId: String, traits: JsonObject) {
        val output = buildJsonObject {
            put("type", "group")
            put("groupId", groupId)
            put("traits", traits)
        }
        println(output)
    }

    fun screen(screenName: String, category: String, properties: JsonObject) {
        val output = buildJsonObject {
            put("type", "screen")
            put("screenName", screenName)
            put("category", category)
            put("properties", properties)
        }
        println(output)
    }
}