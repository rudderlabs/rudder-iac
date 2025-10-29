plugins {
    kotlin("jvm") version "1.9.22"
    application
}

group = "com.example"
version = "1.0.0"

repositories {
    mavenCentral()
}

dependencies {
    implementation("com.rudderstack.sdk.kotlin:core:1.1.0")
}

application {
    mainClass.set("RudderTyperKotlinValidatorKt")
}

tasks.jar {
    manifest {
        attributes["Main-Class"] = "RudderTyperKotlinValidatorKt"
    }
    from(configurations.runtimeClasspath.get().map { if (it.isDirectory) it else zipTree(it) })
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
}

kotlin {
    jvmToolchain(17)
}