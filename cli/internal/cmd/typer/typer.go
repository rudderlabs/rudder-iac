package typer

import (
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/spf13/cobra"
)

// NewCmdTyper creates the typer command
func NewCmdTyper() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "typer",
		Short: "Generate platform-specific RudderAnalytics bindings",
		Long:  "Generate platform-specific RudderAnalytics bindings from tracking plans",
		Example: `  rudder-cli typer generate --output ./generated
  rudder-cli typer generate --output ./src/main/kotlin --platform kotlin`,
	}

	cmd.AddCommand(newCmdGenerate())

	return cmd
}

// newCmdGenerate creates the generate subcommand
func newCmdGenerate() *cobra.Command {
	var (
		outputDir string
		platform  string
		planFile  string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code files from a tracking plan",
		Long:  "Generate platform-specific code files from a tracking plan using the FileManager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(outputDir, platform, planFile)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./generated", "Output directory for generated files")
	cmd.Flags().StringVarP(&platform, "platform", "p", "kotlin", "Target platform (kotlin, swift, etc.)")
	cmd.Flags().StringVarP(&planFile, "plan", "f", "", "Path to tracking plan file (optional, for demo purposes)")

	// Mark required flags
	cmd.MarkFlagRequired("output")

	return cmd
}

// runGenerate executes the code generation process
func runGenerate(outputDir, platform, planFile string) error {
	// Create FileManager
	fileManager, err := core.NewFileManager(outputDir)
	if err != nil {
		return fmt.Errorf("creating file manager: %w", err)
	}

	fmt.Printf("✅ FileManager created successfully\n")
	fmt.Printf("📁 Output directory: %s\n", fileManager.GetOutputDir())

	// Generate sample files for demonstration
	files, err := generateSampleFiles(platform)
	if err != nil {
		return fmt.Errorf("generating sample files: %w", err)
	}

	// Write files using FileManager
	if err := fileManager.WriteFiles(files); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	fmt.Printf("✅ Successfully generated %d files\n", len(files))

	// List generated files
	for _, file := range files {
		fullPath := filepath.Join(outputDir, file.Path)
		fmt.Printf("📄 %s\n", fullPath)
	}

	return nil
}

// generateSampleFiles creates sample files for demonstration
func generateSampleFiles(platform string) ([]core.File, error) {
	switch platform {
	case "kotlin":
		return generateKotlinFiles(), nil
	case "swift":
		return generateSwiftFiles(), nil
	default:
		return generateGenericFiles(platform), nil
	}
}

// generateKotlinFiles creates sample Kotlin files
func generateKotlinFiles() []core.File {
	return []core.File{
		{
			Path: "com/rudderstack/analytics/RudderAnalytics.kt",
			Content: `package com.rudderstack.analytics

import kotlinx.serialization.Serializable

/**
 * RudderAnalytics - Main entry point for analytics tracking
 */
@Serializable
data class RudderAnalytics(
    val userId: String,
    val email: String? = null,
    val properties: Map<String, Any> = emptyMap()
) {
    companion object {
        fun track(eventName: String, properties: Map<String, Any> = emptyMap()) {
            // Implementation would go here
            println("Tracking event: $eventName")
        }
        
        fun identify(userId: String, traits: Map<String, Any> = emptyMap()) {
            // Implementation would go here
            println("Identifying user: $userId")
        }
    }
}`,
		},
		{
			Path: "com/rudderstack/analytics/Events.kt",
			Content: `package com.rudderstack.analytics

import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName

/**
 * Generated event types for tracking
 */
@Serializable
data class UserSignedUpEvent(
    @SerialName("userId")
    val userId: String,
    
    @SerialName("email")
    val email: String? = null,
    
    @SerialName("source")
    val source: String? = null
)

@Serializable
data class ProductViewedEvent(
    @SerialName("productId")
    val productId: String,
    
    @SerialName("productName")
    val productName: String,
    
    @SerialName("category")
    val category: String? = null
)`,
		},
		{
			Path: "com/rudderstack/analytics/Types.kt",
			Content: `package com.rudderstack.analytics

import kotlinx.serialization.Serializable

/**
 * Custom types for analytics
 */
@Serializable
data class UserAddress(
    val street: String,
    val city: String,
    val country: String,
    val postalCode: String? = null
)

@Serializable
data class ProductDetails(
    val id: String,
    val name: String,
    val price: Double,
    val category: String? = null
)`,
		},
	}
}

// generateSwiftFiles creates sample Swift files
func generateSwiftFiles() []core.File {
	return []core.File{
		{
			Path: "RudderAnalytics/RudderAnalytics.swift",
			Content: `import Foundation

/**
 * RudderAnalytics - Main entry point for analytics tracking
 */
public class RudderAnalytics {
    public let userId: String
    public let email: String?
    public let properties: [String: Any]
    
    public init(userId: String, email: String? = nil, properties: [String: Any] = [:]) {
        self.userId = userId
        self.email = email
        self.properties = properties
    }
    
    public static func track(eventName: String, properties: [String: Any] = [:]) {
        // Implementation would go here
        print("Tracking event: \\(eventName)")
    }
    
    public static func identify(userId: String, traits: [String: Any] = [:]) {
        // Implementation would go here
        print("Identifying user: \\(userId)")
    }
}`,
		},
		{
			Path: "RudderAnalytics/Events.swift",
			Content: `import Foundation

/**
 * Generated event types for tracking
 */
public struct UserSignedUpEvent: Codable {
    public let userId: String
    public let email: String?
    public let source: String?
    
    public init(userId: String, email: String? = nil, source: String? = nil) {
        self.userId = userId
        self.email = email
        self.source = source
    }
}

public struct ProductViewedEvent: Codable {
    public let productId: String
    public let productName: String
    public let category: String?
    
    public init(productId: String, productName: String, category: String? = nil) {
        self.productId = productId
        self.productName = productName
        self.category = category
    }
}`,
		},
	}
}

// generateGenericFiles creates generic files for any platform
func generateGenericFiles(platform string) []core.File {
	return []core.File{
		{
			Path: fmt.Sprintf("%s/README.md", platform),
			Content: fmt.Sprintf(`# %s Generated Code

This directory contains generated code for the %s platform.

## Files Generated

- Main analytics class
- Event type definitions
- Custom type definitions

## Usage

The generated code provides type-safe analytics tracking for your %s application.

Generated by RudderTyper 2.0
`, platform, platform, platform),
		},
		{
			Path: fmt.Sprintf("%s/analytics.%s", platform, getFileExtension(platform)),
			Content: fmt.Sprintf(`// Generated %s analytics code
// Platform: %s
// Generated by RudderTyper 2.0

// Main analytics class
class RudderAnalytics {
    constructor(userId, email = null, properties = {}) {
        this.userId = userId;
        this.email = email;
        this.properties = properties;
    }
    
    static track(eventName, properties = {}) {
        console.log('Tracking event:', eventName);
    }
    
    static identify(userId, traits = {}) {
        console.log('Identifying user:', userId);
    }
}

module.exports = RudderAnalytics;
`, platform, platform),
		},
	}
}

// getFileExtension returns the appropriate file extension for a platform
func getFileExtension(platform string) string {
	switch platform {
	case "javascript", "js":
		return "js"
	case "typescript", "ts":
		return "ts"
	case "python", "py":
		return "py"
	case "java":
		return "java"
	default:
		return "txt"
	}
}
