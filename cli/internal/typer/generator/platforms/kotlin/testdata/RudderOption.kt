package com.rudderstack.sdk.kotlin.core.internals.models

import kotlinx.serialization.json.JsonObject

data class RudderOption(
    val customContext: JsonObject
)