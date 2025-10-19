package com.rudderstack.ruddertyper

import com.rudderstack.sdk.kotlin.core.Analytics
import com.rudderstack.sdk.kotlin.core.internals.models.RudderOption
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerializationException
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.descriptors.buildClassSerialDescriptor
import kotlinx.serialization.encoding.Encoder
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonEncoder
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject

/** Whether user is active */
typealias CustomTypeActive = Boolean

/** User's age in years */
typealias CustomTypeAge = Double

/** Custom type for email validation */
typealias CustomTypeEmail = String

/** List of email addresses */
typealias CustomTypeEmailList = List<CustomTypeEmail>

/** Empty object that allows additional properties */
typealias CustomTypeEmptyObjectWithAdditionalProps = JsonObject

/** List of user profiles */
typealias CustomTypeProfileList = List<CustomTypeUserProfile>

/** User active status */
typealias PropertyActive = CustomTypeActive

/** User's age */
typealias PropertyAge = CustomTypeAge

/** An array that can contain any type of items */
typealias PropertyArrayOfAny = List<JsonElement>

/** Array of user contacts */
typealias PropertyContacts = List<CustomTypeEmail>

/** example of object property */
typealias PropertyContext = JsonObject

/** User's email address */
typealias PropertyEmail = CustomTypeEmail

/** User's email addresses */
typealias PropertyEmailList = CustomTypeEmailList

/** Property with empty object allowing additional properties */
typealias PropertyEmptyObjectWithAdditionalProps = CustomTypeEmptyObjectWithAdditionalProps

/** Feature configuration information */
typealias PropertyFeatureConfig = CustomTypeFeatureConfig

/** User's first name */
typealias PropertyFirstName = String

/** IP address of the user */
typealias PropertyIpAddress = String

/** User's last name */
typealias PropertyLastName = String

/** An array with items that can be string or integer */
typealias PropertyMultiTypeArray = List<ArrayItemMultiTypeArray>

/** demonstrates multiple levels of nesting */
typealias PropertyNestedContext = JsonObject

/** Nested property with empty object allowing additional properties */
typealias PropertyNestedEmptyObject = JsonObject

/** An object field with no defined structure */
typealias PropertyObjectProperty = JsonObject

/** Page context information */
typealias PropertyPageContext = CustomTypePageContext

/** Additional page data */
typealias PropertyPageData = JsonObject

/** Type of page */
typealias PropertyPageType = String

/** Product identifier */
typealias PropertyProductId = String

/** User profile data */
typealias PropertyProfile = CustomTypeUserProfile

/** List of related user profiles */
typealias PropertyProfileList = CustomTypeProfileList

/** A field that can contain any type of value */
typealias PropertyPropertyOfAny = JsonElement

/** Search query */
typealias PropertyQuery = String

/** User account status */
typealias PropertyStatus = CustomTypeStatus

/** User tags as array of strings */
typealias PropertyTags = List<String>

/** An array with no explicit item type (treated as any) */
typealias PropertyUntypedArray = List<JsonElement>

/** A field with no explicit type (treated as any) */
typealias PropertyUntypedField = JsonElement

/** User access information */
typealias PropertyUserAccess = CustomTypeUserAccess

/** User status enum */
@Serializable
enum class CustomTypeStatus {
    @SerialName("pending")
    PENDING,
    @SerialName("active")
    ACTIVE,
    @SerialName("suspended")
    SUSPENDED,
    @SerialName("deleted")
    DELETED
}

/** Type of device */
@Serializable
enum class PropertyDeviceType {
    @SerialName("mobile")
    MOBILE,
    @SerialName("tablet")
    TABLET,
    @SerialName("desktop")
    DESKTOP,
    @SerialName("smartTV")
    SMART_TV,
    @SerialName("IoT-Device")
    IO_T_DEVICE
}

abstract class SealedClassWithJson {
    abstract val _jsonElement: JsonElement
}

open class SealedClassJsonSerializer<T : SealedClassWithJson> : KSerializer<T> {
    override val descriptor: SerialDescriptor = buildClassSerialDescriptor("SealedClass")

    override fun serialize(encoder: Encoder, value: T) {
        val jsonEncoder = encoder as? JsonEncoder
            ?: throw SerializationException("This serializer only works with JSON")
        jsonEncoder.encodeJsonElement(value._jsonElement)
    }

    override fun deserialize(decoder: Decoder): T {
        throw NotImplementedError("Deserialization is not supported")
    }
}

/** Feature configuration with variants based on multi-type flag */
@Serializable(with = RudderCustomTypeFeatureConfigSerializer::class)
sealed class CustomTypeFeatureConfig : SealedClassWithJson() {
    /** Feature flag that can be boolean or string */
    @SerialName("feature_flag")
    abstract val featureFlag: PropertyFeatureFlag
    abstract override val _jsonElement: JsonElement

    /** Feature enabled (boolean true) */
    @Serializable
    data class Case_True(
        /** User's email address */
        @SerialName("email")
        val email: PropertyEmail
    ) : CustomTypeFeatureConfig() {
        /** Feature flag that can be boolean or string */
        @SerialName("feature_flag")
        override val featureFlag: PropertyFeatureFlag = PropertyFeatureFlag.BooleanValue(true)
        override val _jsonElement: JsonElement = buildJsonObject {
            put("email", Json.encodeToJsonElement(email))
            put("feature_flag", Json.encodeToJsonElement(featureFlag))
        }
    }

    /** Feature disabled (boolean false) */
    @Serializable
    data class Case_False(
        /** User account status */
        @SerialName("status")
        val status: PropertyStatus
    ) : CustomTypeFeatureConfig() {
        /** Feature flag that can be boolean or string */
        @SerialName("feature_flag")
        override val featureFlag: PropertyFeatureFlag = PropertyFeatureFlag.BooleanValue(false)
        override val _jsonElement: JsonElement = buildJsonObject {
            put("status", Json.encodeToJsonElement(status))
            put("feature_flag", Json.encodeToJsonElement(featureFlag))
        }
    }

    /** Feature in beta (string 'beta') */
    @Serializable
    data class CaseBeta(
        /** User tags as array of strings */
        @SerialName("tags")
        val tags: PropertyTags
    ) : CustomTypeFeatureConfig() {
        /** Feature flag that can be boolean or string */
        @SerialName("feature_flag")
        override val featureFlag: PropertyFeatureFlag = PropertyFeatureFlag.StringValue("beta")
        override val _jsonElement: JsonElement = buildJsonObject {
            put("tags", Json.encodeToJsonElement(tags))
            put("feature_flag", Json.encodeToJsonElement(featureFlag))
        }
    }
}

private object RudderCustomTypeFeatureConfigSerializer : SealedClassJsonSerializer<CustomTypeFeatureConfig>()

/** Page context with variants based on page type */
@Serializable(with = RudderCustomTypePageContextSerializer::class)
sealed class CustomTypePageContext : SealedClassWithJson() {
    /** Type of page */
    @SerialName("page_type")
    abstract val pageType: PropertyPageType
    abstract override val _jsonElement: JsonElement

    /** Search page variant */
    @Serializable
    data class CaseSearch(
        /** Search query */
        @SerialName("query")
        val query: PropertyQuery
    ) : CustomTypePageContext() {
        /** Type of page */
        @SerialName("page_type")
        override val pageType: PropertyPageType = "search"
        override val _jsonElement: JsonElement = buildJsonObject {
            put("query", Json.encodeToJsonElement(query))
            put("page_type", Json.encodeToJsonElement(pageType))
        }
    }

    /** Product page variant */
    @Serializable
    data class CaseProduct(
        /** Product identifier */
        @SerialName("product_id")
        val productId: PropertyProductId
    ) : CustomTypePageContext() {
        /** Type of page */
        @SerialName("page_type")
        override val pageType: PropertyPageType = "product"
        override val _jsonElement: JsonElement = buildJsonObject {
            put("product_id", Json.encodeToJsonElement(productId))
            put("page_type", Json.encodeToJsonElement(pageType))
        }
    }

    /** Default case */
    @Serializable
    data class Default(
        /** Additional page data */
        @SerialName("page_data")
        val pageData: PropertyPageData? = null,

        /** Type of page */
        @SerialName("page_type")
        override val pageType: PropertyPageType
    ) : CustomTypePageContext() {
        override val _jsonElement: JsonElement = buildJsonObject {
            pageData?.let { put("page_data", Json.encodeToJsonElement(it)) }
            put("page_type", Json.encodeToJsonElement(pageType))
        }
    }
}

private object RudderCustomTypePageContextSerializer : SealedClassJsonSerializer<CustomTypePageContext>()

/** User access with variants based on active status */
@Serializable(with = RudderCustomTypeUserAccessSerializer::class)
sealed class CustomTypeUserAccess : SealedClassWithJson() {
    /** User active status */
    @SerialName("active")
    abstract val active: PropertyActive
    abstract override val _jsonElement: JsonElement

    /** Active user access */
    @Serializable
    data class Case_True(
        /** User's email address */
        @SerialName("email")
        val email: PropertyEmail
    ) : CustomTypeUserAccess() {
        /** User active status */
        @SerialName("active")
        override val active: PropertyActive = true
        override val _jsonElement: JsonElement = buildJsonObject {
            put("email", Json.encodeToJsonElement(email))
            put("active", Json.encodeToJsonElement(active))
        }
    }

    /** Inactive user access */
    @Serializable
    data class Case_False(
        /** User account status */
        @SerialName("status")
        val status: PropertyStatus
    ) : CustomTypeUserAccess() {
        /** User active status */
        @SerialName("active")
        override val active: PropertyActive = false
        override val _jsonElement: JsonElement = buildJsonObject {
            put("status", Json.encodeToJsonElement(status))
            put("active", Json.encodeToJsonElement(active))
        }
    }
}

private object RudderCustomTypeUserAccessSerializer : SealedClassJsonSerializer<CustomTypeUserAccess>()

/** Feature flag that can be boolean or string */
@Serializable(with = RudderPropertyFeatureFlagSerializer::class)
sealed class PropertyFeatureFlag : SealedClassWithJson() {
    abstract override val _jsonElement: JsonElement
    /** Represents a 'boolean' value */
    @Serializable
    data class BooleanValue(
        @SerialName("value")
        val value: Boolean
    ) : PropertyFeatureFlag() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }

    /** Represents a 'string' value */
    @Serializable
    data class StringValue(
        @SerialName("value")
        val value: String
    ) : PropertyFeatureFlag() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }
}

private object RudderPropertyFeatureFlagSerializer : SealedClassJsonSerializer<PropertyFeatureFlag>()

/** Item type for multi_type_array array */
@Serializable(with = RudderArrayItemMultiTypeArraySerializer::class)
sealed class ArrayItemMultiTypeArray : SealedClassWithJson() {
    abstract override val _jsonElement: JsonElement
    /** Represents a 'string' value */
    @Serializable
    data class StringValue(
        @SerialName("value")
        val value: String
    ) : ArrayItemMultiTypeArray() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }

    /** Represents a 'integer' value */
    @Serializable
    data class IntegerValue(
        @SerialName("value")
        val value: Long
    ) : ArrayItemMultiTypeArray() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }
}

private object RudderArrayItemMultiTypeArraySerializer : SealedClassJsonSerializer<ArrayItemMultiTypeArray>()

/** A field that can be string, integer, or boolean */
@Serializable(with = RudderPropertyMultiTypeFieldSerializer::class)
sealed class PropertyMultiTypeField : SealedClassWithJson() {
    abstract override val _jsonElement: JsonElement
    /** Represents a 'string' value */
    @Serializable
    data class StringValue(
        @SerialName("value")
        val value: String
    ) : PropertyMultiTypeField() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }

    /** Represents a 'integer' value */
    @Serializable
    data class IntegerValue(
        @SerialName("value")
        val value: Long
    ) : PropertyMultiTypeField() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }

    /** Represents a 'boolean' value */
    @Serializable
    data class BooleanValue(
        @SerialName("value")
        val value: Boolean
    ) : PropertyMultiTypeField() {

        override val _jsonElement: JsonElement = JsonPrimitive(value)
    }
}

private object RudderPropertyMultiTypeFieldSerializer : SealedClassJsonSerializer<PropertyMultiTypeField>()

/** Example event to demonstrate variants */
@Serializable(with = RudderTrackEventWithVariantsPropertiesSerializer::class)
sealed class TrackEventWithVariantsProperties : SealedClassWithJson() {
    /** Type of device */
    @SerialName("device_type")
    abstract val deviceType: PropertyDeviceType
    abstract override val _jsonElement: JsonElement

    /** Mobile device page view */
    @Serializable
    data class CaseMobile(
        /** Page context information */
        @SerialName("page_context")
        val pageContext: PropertyPageContext? = null,

        /** User profile data */
        @SerialName("profile")
        val profile: PropertyProfile,

        /** User tags as array of strings */
        @SerialName("tags")
        val tags: PropertyTags
    ) : TrackEventWithVariantsProperties() {
        /** Type of device */
        @SerialName("device_type")
        override val deviceType: PropertyDeviceType = PropertyDeviceType.MOBILE
        override val _jsonElement: JsonElement = buildJsonObject {
            pageContext?.let { put("page_context", Json.encodeToJsonElement(it)) }
            put("profile", Json.encodeToJsonElement(profile))
            put("tags", Json.encodeToJsonElement(tags))
            put("device_type", Json.encodeToJsonElement(deviceType))
        }
    }

    /** Desktop page view */
    @Serializable
    data class CaseDesktop(
        /** User's first name */
        @SerialName("first_name")
        val firstName: PropertyFirstName,

        /** User's last name */
        @SerialName("last_name")
        val lastName: PropertyLastName? = null,

        /** Page context information */
        @SerialName("page_context")
        val pageContext: PropertyPageContext? = null,

        /** User profile data */
        @SerialName("profile")
        val profile: PropertyProfile
    ) : TrackEventWithVariantsProperties() {
        /** Type of device */
        @SerialName("device_type")
        override val deviceType: PropertyDeviceType = PropertyDeviceType.DESKTOP
        override val _jsonElement: JsonElement = buildJsonObject {
            put("first_name", Json.encodeToJsonElement(firstName))
            lastName?.let { put("last_name", Json.encodeToJsonElement(it)) }
            pageContext?.let { put("page_context", Json.encodeToJsonElement(it)) }
            put("profile", Json.encodeToJsonElement(profile))
            put("device_type", Json.encodeToJsonElement(deviceType))
        }
    }

    /** Default case */
    @Serializable
    data class Default(
        /** Type of device */
        @SerialName("device_type")
        override val deviceType: PropertyDeviceType,

        /** Page context information */
        @SerialName("page_context")
        val pageContext: PropertyPageContext? = null,

        /** User profile data */
        @SerialName("profile")
        val profile: PropertyProfile,

        /** A field with no explicit type (treated as any) */
        @SerialName("untyped_field")
        val untypedField: PropertyUntypedField? = null
    ) : TrackEventWithVariantsProperties() {
        override val _jsonElement: JsonElement = buildJsonObject {
            put("device_type", Json.encodeToJsonElement(deviceType))
            pageContext?.let { put("page_context", Json.encodeToJsonElement(it)) }
            put("profile", Json.encodeToJsonElement(profile))
            untypedField?.let { put("untyped_field", Json.encodeToJsonElement(it)) }
        }
    }
}

private object RudderTrackEventWithVariantsPropertiesSerializer : SealedClassJsonSerializer<TrackEventWithVariantsProperties>()

/** User profile information */
@Serializable
data class CustomTypeUserProfile(
    /** User's email address */
    @SerialName("email")
    val email: PropertyEmail,

    /** User's first name */
    @SerialName("first_name")
    val firstName: PropertyFirstName,

    /** User's last name */
    @SerialName("last_name")
    val lastName: PropertyLastName? = null
)

/** Group association event */
@Serializable
data class GroupTraits(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive
)

/** User identification event */
@Serializable
data class IdentifyTraits(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive? = null,

    /** User's email address */
    @SerialName("email")
    val email: PropertyEmail
)

/** Page view event */
@Serializable
data class PageProperties(
    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile
)

/** Screen view event */
@Serializable
data class ScreenProperties(
    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile? = null
)

/** Triggered when a user signs up */
@Serializable
data class TrackUserSignedUpProperties(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive,

    /** User's age */
    @SerialName("age")
    val age: PropertyAge? = null,

    /** An array that can contain any type of items */
    @SerialName("array_of_any")
    val arrayOfAny: PropertyArrayOfAny? = null,

    /** Array of user contacts */
    @SerialName("contacts")
    val contacts: PropertyContacts? = null,

    /** example of object property */
    @SerialName("context")
    val context: TrackUserSignedUpProperties.Context? = null,

    /** Type of device */
    @SerialName("device_type")
    val deviceType: PropertyDeviceType? = null,

    /** User's email addresses */
    @SerialName("email_list")
    val emailList: PropertyEmailList? = null,

    /** Property with empty object allowing additional properties */
    @SerialName("empty_object_with_additional_props")
    val emptyObjectWithAdditionalProps: PropertyEmptyObjectWithAdditionalProps? = null,

    /** Feature configuration information */
    @SerialName("feature_config")
    val featureConfig: PropertyFeatureConfig? = null,

    /** An array with items that can be string or integer */
    @SerialName("multi_type_array")
    val multiTypeArray: PropertyMultiTypeArray? = null,

    /** A field that can be string, integer, or boolean */
    @SerialName("multi_type_field")
    val multiTypeField: PropertyMultiTypeField? = null,

    /** Nested property with empty object allowing additional properties */
    @SerialName("nested_empty_object")
    val nestedEmptyObject: PropertyNestedEmptyObject? = null,

    /** An object field with no defined structure */
    @SerialName("object_property")
    val objectProperty: PropertyObjectProperty? = null,

    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile,

    /** List of related user profiles */
    @SerialName("profile_list")
    val profileList: PropertyProfileList? = null,

    /** A field that can contain any type of value */
    @SerialName("property_of_any")
    val propertyOfAny: PropertyPropertyOfAny? = null,

    /** User account status */
    @SerialName("status")
    val status: PropertyStatus? = null,

    /** User tags as array of strings */
    @SerialName("tags")
    val tags: PropertyTags? = null,

    /** An array with no explicit item type (treated as any) */
    @SerialName("untyped_array")
    val untypedArray: PropertyUntypedArray? = null,

    /** A field with no explicit type (treated as any) */
    @SerialName("untyped_field")
    val untypedField: PropertyUntypedField? = null,

    /** User access information */
    @SerialName("user_access")
    val userAccess: PropertyUserAccess? = null
) {
    /** example of object property */
    @Serializable
    data class Context(
        /** IP address of the user */
        @SerialName("ip_address")
        val ipAddress: PropertyIpAddress,

        /** demonstrates multiple levels of nesting */
        @SerialName("nested_context")
        val nestedContext: TrackUserSignedUpProperties.Context.NestedContext
    ) {
        /** demonstrates multiple levels of nesting */
        @Serializable
        data class NestedContext(
            /** User profile data */
            @SerialName("profile")
            val profile: PropertyProfile? = null
        )
    }
}

class RudderAnalytics(private val analytics: Analytics) {
    private val json = Json {
        prettyPrint = true
        encodeDefaults = false
    }

    private val context = buildJsonObject {
        put("ruddertyper", buildJsonObject {
            put("platform", "kotlin")
            put("rudderCLIVersion", "1.0.0")
            put("trackingPlanId", "plan_12345")
            put("trackingPlanVersion", 13)
        })
    }

    /**
     * Group association event
     */
    fun group(groupId: String, traits: GroupTraits) {
        analytics.group(
            groupId = groupId,
            traits = json.encodeToJsonElement(traits).jsonObject,
            options = RudderOption(customContext = context)
        )
    }

    /**
     * User identification event
     */
    fun identify(userId: String = "", traits: IdentifyTraits) {
        analytics.identify(
            userId = userId,
            traits = json.encodeToJsonElement(traits).jsonObject,
            options = RudderOption(customContext = context)
        )
    }

    /**
     * Screen view event
     */
    fun screen(screenName: String, category: String = "", properties: ScreenProperties) {
        analytics.screen(
            screenName = screenName,
            category = category,
            properties = json.encodeToJsonElement(properties).jsonObject,
            options = RudderOption(customContext = context)
        )
    }

    /**
     * Example event to demonstrate variants
     */
    fun trackEventWithVariants(properties: TrackEventWithVariantsProperties) {
        analytics.track(
            name = "Event With Variants",
            properties = json.encodeToJsonElement(properties).jsonObject,
            options = RudderOption(customContext = context)
        )
    }

    /**
     * Triggered when a user signs up
     */
    fun trackUserSignedUp(properties: TrackUserSignedUpProperties) {
        analytics.track(
            name = "User Signed Up",
            properties = json.encodeToJsonElement(properties).jsonObject,
            options = RudderOption(customContext = context)
        )
    }
}