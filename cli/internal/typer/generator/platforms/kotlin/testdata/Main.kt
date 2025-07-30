package com.rudderstack.ruddertyper

import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName

/** Whether user is active */
typealias CustomTypeActive = Boolean

/** User's age in years */
typealias CustomTypeAge = Double

/** Custom type for email validation */
typealias CustomTypeEmail = String

/** User active status */
typealias PropertyActive = CustomTypeActive

/** User's age */
typealias PropertyAge = CustomTypeAge

/** User's email address */
typealias PropertyEmail = CustomTypeEmail

/** User's first name */
typealias PropertyFirstName = String

/** User's last name */
typealias PropertyLastName = String

/** User profile data */
typealias PropertyProfile = CustomTypeUserProfile

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
    val lastName: PropertyLastName?
)
