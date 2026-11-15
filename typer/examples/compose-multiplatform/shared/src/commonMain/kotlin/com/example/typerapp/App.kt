package com.example.typerapp

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeContentPadding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.example.typerapp.typer.PropertyTpverE2ePlanTier
import com.example.typerapp.typer.RudderTyper
import com.example.typerapp.typer.RudderTyperTransport
import com.example.typerapp.typer.TrackTpverE2ePlanChangedProperties
import com.example.typerapp.typer.TrackTpverE2eSeatsAddedProperties

/** The tracking plan of each workspace. Plan IDs and versions differ per workspace, the event models do not. */
enum class Environment(val trackingPlanId: String, val trackingPlanVersion: Int) {
    Production("tp_production_plan", 7),
    Staging("tp_staging_plan", 9),
}

@Composable
fun App(transport: RudderTyperTransport) {
    var environment by remember { mutableStateOf(Environment.Production) }
    // Switching environment swaps the plan the events are stamped with, the way the app would also
    // swap the data plane URL. The generated models stay the same.
    val typer = remember(environment) {
        RudderTyper(transport, trackingPlanId = environment.trackingPlanId, trackingPlanVersion = environment.trackingPlanVersion)
    }

    MaterialTheme {
        Column(
            modifier = Modifier.fillMaxSize().safeContentPadding().padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text("RudderTyper on Compose Multiplatform", style = MaterialTheme.typography.titleLarge)
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Staging")
                Switch(
                    checked = environment == Environment.Staging,
                    onCheckedChange = { environment = if (it) Environment.Staging else Environment.Production },
                )
            }
            Button(onClick = {
                typer.trackTpverE2ePlanChanged(TrackTpverE2ePlanChangedProperties(tpverE2ePlanTier = PropertyTpverE2ePlanTier.PRO))
            }) { Text("Track: plan changed") }
            Button(onClick = {
                typer.trackTpverE2eSeatsAdded(TrackTpverE2eSeatsAddedProperties(tpverE2eSeatCount = 3))
            }) { Text("Track: seats added") }
        }
    }
}
