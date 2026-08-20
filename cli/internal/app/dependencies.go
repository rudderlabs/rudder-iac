package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	accountsProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	dgProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	destProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	attentivetag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/attentive_tag"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bq"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bqstream"
	confluentcloud "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/confluent_cloud"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	facebookconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gcs"
	googlepubsub "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlepubsub"
	googlesheets "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlesheets"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/kinesis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/marketo"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/postgres"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/redis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/rs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	s3datalake "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3_datalake"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/salesforce"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/slack"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowflake"
	snowpipestreaming "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowpipe_streaming"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/statsig"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/zendesk"
	esProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/workspace"
	"github.com/rudderlabs/rudder-iac/cli/internal/ruledoc"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var (
	v string
)

// Providers holds instances of all providers used in the application
// Provider types are intentionally set to specific provider implementations
// instead of the generic provider.Provider interface to allow access to
// provider-specific methods if needed.
type Providers struct {
	DataCatalog     *datacatalog.Provider
	RETL            *retl.Provider
	EventStream     *esProvider.Provider
	Transformations *transformations.Provider
	Workspace       *workspace.Provider
	DataGraph       *dgProvider.Provider
	Destination     *destProvider.Provider
	Account         *accountsProvider.Provider
}

type deps struct {
	client            *client.Client
	providers         *Providers
	compositeProvider provider.Provider
}

// Deps defines the dependencies initialized globally for Rudder CLI
type Deps interface {
	// Client returns the RudderStack API client instance, configured with authentication and base URL
	Client() *client.Client

	// Providers returns the initialized Providers struct containing all provider instances
	// used in the application when individual provider access is needed.
	Providers() *Providers

	// CompositeProvider returns a composite provider aggregating all individual providers
	// used by components that operate across multiple providers.
	CompositeProvider() provider.Provider

	// Registry builds a validation rule registry from the composite provider so
	// the docs generator observes the same rule set as project validation.
	Registry() (rules.Registry, error)

	// NewProject creates a new project instance with the composite provider.
	NewProject(opts ...project.ProjectOption) project.Project

	// NewDataCatalogProject creates a new project instance with only the DataCatalog provider.
	// Used by trackingplan commands.
	NewDataCatalogProject() project.Project
}

func Initialise(version string) {
	v = version
}

func validateDependencies() error {
	cfg := config.GetConfig()
	if cfg.Auth.AccessToken == "" {
		return fmt.Errorf("access token is required, please run `rudder-cli auth login`, or set the access token via the RUDDERSTACK_ACCESS_TOKEN environment variable")
	}

	return nil
}

func NewDeps() (Deps, error) {
	if err := validateDependencies(); err != nil {
		return nil, err
	}

	c, err := setupClient(v)
	if err != nil {
		return nil, fmt.Errorf("setup client: %w", err)
	}

	cp, p, err := composeProviders(c)
	if err != nil {
		return nil, err
	}

	return &deps{
		client:            c,
		providers:         p,
		compositeProvider: cp,
	}, nil
}

// GenerateRuleCatalog composes the same providers project validation uses and
// hands them to ruledoc.Build, which joins the live rules with the authored
// *.docs.yaml fragments and returns the validated catalog.
//
// Composition is the only part that needs the app's machinery, so it lives
// here in the composition root; the provider-in, catalog-out assembly lives in
// package ruledoc where it can be tested without credentials. Generation makes
// no network calls, so it skips the access-token check NewDeps enforces.
//
// verrs carries catalog validation failures (e.g. a registered rule with no
// authored fragment); a non-nil error means the catalog could not be assembled
// at all. generatedAt is injected so callers own the timestamp.
func GenerateRuleCatalog(generatedAt string) (docs.DocumentedRules, []error, error) {
	cp, err := newCompositeProvider()
	if err != nil {
		return docs.DocumentedRules{}, nil, fmt.Errorf("building composite provider: %w", err)
	}

	return ruledoc.Build(cp, GetVersion(), generatedAt)
}

// newCompositeProvider builds the composite provider without requiring
// credentials. It is used only by GenerateRuleCatalog: rule-doc generation
// enumerates rules and reads authored fragments but makes no network calls, so
// it skips the auth check NewDeps enforces and feeds client.New a placeholder
// token (an empty token is rejected outright, which would otherwise break
// generation in CI where no credentials are configured). It shares
// composeProviders with NewDeps so the documented rule set stays identical to
// the one project validation observes — they can't drift.
func newCompositeProvider() (provider.Provider, error) {
	cfg := config.GetConfig()

	c, err := client.New(
		"rule-doc-generation", // unused: generation makes no API calls
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+v),
	)
	if err != nil {
		return nil, fmt.Errorf("setup client: %w", err)
	}

	cp, _, err := composeProviders(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize composite provider: %w", err)
	}

	return cp, nil
}

// composeProviders builds the provider set and aggregates it into a composite
// provider. Shared by NewDeps and newCompositeProvider so every consumer
// observes the same providers (and therefore the same registered rules).
func composeProviders(c *client.Client) (provider.Provider, *Providers, error) {
	rawProviders, providerMap, err := setupProviders(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	cp, err := provider.NewCompositeProvider(providerMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize composite provider: %w", err)
	}

	return cp, rawProviders, nil
}

func setupClient(version string) (*client.Client, error) {
	cfg := config.GetConfig()
	return client.New(
		cfg.Auth.AccessToken,
		client.WithBaseURL(cfg.APIURL),
		client.WithUserAgent("rudder-cli/"+version),
	)
}

func setupProviders(c *client.Client) (*Providers, map[string]provider.Provider, error) {
	cfg := config.GetConfig()

	catalogClient, err := catalog.NewRudderDataCatalog(
		c,
		catalog.WithConcurrency(cfg.Concurrency.CatalogClient),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize data catalog client: %w", err)
	}

	dcp := datacatalog.New(catalogClient)
	retlp := retl.New(retlClient.NewRudderRETLStore(c))
	esp := esProvider.New(esClient.NewRudderEventStreamStore(c))
	trp := transformations.NewProvider(c)
	wsp := workspace.New(c)
	dgp := dgProvider.NewProvider(dgClient.NewRudderDataGraphClient(c), c.Accounts)

	providers := &Providers{
		DataCatalog:     dcp,
		RETL:            retlp,
		EventStream:     esp,
		Transformations: trp,
		Workspace:       wsp,
		DataGraph:       dgp,
	}

	providerMap := map[string]provider.Provider{
		"datacatalog":     dcp,
		"retl":            retlp,
		"eventstream":     esp,
		"transformations": trp,
		"datagraph":       dgp,
	}

	if cfg.ExperimentalFlags.DestinationSupport {
		destRegistry, err := newDestinationRegistry(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize destination registry: %w", err)
		}
		dp := destProvider.NewProvider(c, destRegistry)

		providerMap["destination"] = dp
		providers.Destination = dp

	}

	if cfg.ExperimentalFlags.AccountSupport {
		ap := accountsProvider.NewProvider(c.Accounts)

		providerMap["account"] = ap
		providers.Account = ap
	}

	return providers, providerMap, nil
}

// newDestinationRegistry builds the destination definition registry.
// DestinationSupport must be on before any definitions are registered.
// Unverified definitions additionally require UnverifiedDestinations.
func newDestinationRegistry(cfg config.Config) (*definitions.Registry, error) {
	registry := definitions.NewRegistry()
	if !cfg.ExperimentalFlags.DestinationSupport {
		return registry, nil
	}

	if err := registry.Register(s3.NewDefinition()); err != nil {
		return nil, fmt.Errorf("registering s3 destination definition: %w", err)
	}

	if cfg.ExperimentalFlags.UnverifiedDestinations {
		if err := registry.Register(attentivetag.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering attentive_tag destination definition: %w", err)
		}
		if err := registry.Register(bq.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering bq destination definition: %w", err)
		}
		if err := registry.Register(bqstream.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering bqstream destination definition: %w", err)
		}
		if err := registry.Register(confluentcloud.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering confluent_cloud destination definition: %w", err)
		}
		if err := registry.Register(customerioaudience.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering customerio_audience destination definition: %w", err)
		}
		if err := registry.Register(facebookconversions.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering facebook_conversions destination definition: %w", err)
		}
		if err := registry.Register(gcs.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering gcs destination definition: %w", err)
		}
		if err := registry.Register(googlepubsub.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering googlepubsub destination definition: %w", err)
		}
		if err := registry.Register(googlesheets.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering googlesheets destination definition: %w", err)
		}
		if err := registry.Register(httpdest.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering http destination definition: %w", err)
		}
		if err := registry.Register(kinesis.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering kinesis destination definition: %w", err)
		}
		if err := registry.Register(marketo.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering marketo destination definition: %w", err)
		}
		if err := registry.Register(postgres.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering postgres destination definition: %w", err)
		}
		if err := registry.Register(redis.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering redis destination definition: %w", err)
		}
		if err := registry.Register(rs.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering rs destination definition: %w", err)
		}
		if err := registry.Register(s3datalake.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering s3_datalake destination definition: %w", err)
		}
		if err := registry.Register(salesforce.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering salesforce destination definition: %w", err)
		}
		if err := registry.Register(slack.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering slack destination definition: %w", err)
		}
		if err := registry.Register(snowflake.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering snowflake destination definition: %w", err)
		}
		if err := registry.Register(snowpipestreaming.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering snowpipe_streaming destination definition: %w", err)
		}
		if err := registry.Register(statsig.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering statsig destination definition: %w", err)
		}
		if err := registry.Register(zendesk.NewDefinition()); err != nil {
			return nil, fmt.Errorf("registering zendesk destination definition: %w", err)
		}
	}
	return registry, nil
}

func SyncReporter() syncer.SyncReporter {
	if ui.IsTerminal() {
		return &reporters.ProgressSyncReporter{}
	}

	return &reporters.PlainSyncReporter{}
}

func (d *deps) Client() *client.Client {
	return d.client
}

func (d *deps) Providers() *Providers {
	return d.providers
}

func (d *deps) CompositeProvider() provider.Provider {
	return d.compositeProvider
}

// Registry builds a validation rule registry from the composite provider,
// sharing the same construction as project validation so the docs generator
// observes an identical rule set.
func (d *deps) Registry() (rules.Registry, error) {
	return project.BuildRegistry(d.CompositeProvider(), importmanifest.New(), config.GetConfig().ExperimentalFlags.ImportMerge)
}

// NewProject creates a project with composite provider.
func (d *deps) NewProject(opts ...project.ProjectOption) project.Project {
	return project.New(d.CompositeProvider(), opts...)
}

// NewDataCatalogProject creates a project with only the DataCatalog provider.
// Used by trackingplan commands that only need data catalog functionality.
func (d *deps) NewDataCatalogProject() project.Project {
	return project.New(d.Providers().DataCatalog)
}

func GetVersion() string {
	return v
}
