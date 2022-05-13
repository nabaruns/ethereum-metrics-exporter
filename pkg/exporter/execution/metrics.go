package execution

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samcm/ethereum-metrics-exporter/pkg/exporter/execution/api"
	"github.com/samcm/ethereum-metrics-exporter/pkg/exporter/execution/jobs"
	"github.com/sirupsen/logrus"
)

type Metrics interface {
	StartAsync(ctx context.Context)
}

type metrics struct {
	log            logrus.FieldLogger
	syncMetrics    jobs.SyncStatus
	generalMetrics jobs.GeneralMetrics
	txpoolMetrics  jobs.TXPool
	adminMetrics   jobs.Admin

	enabledJobs map[string]bool
}

func NewMetrics(client *ethclient.Client, internalApi api.ExecutionClient, log logrus.FieldLogger, nodeName, namespace string, enabledModules []string) Metrics {
	constLabels := make(prometheus.Labels)
	constLabels["ethereum_role"] = "execution"
	constLabels["node_name"] = nodeName

	m := &metrics{
		log:            log,
		generalMetrics: jobs.NewGeneralMetrics(client, internalApi, log, namespace, constLabels),
		syncMetrics:    jobs.NewSyncStatus(client, internalApi, log, namespace, constLabels),
		txpoolMetrics:  jobs.NewTXPool(client, internalApi, log, namespace, constLabels),
		adminMetrics:   jobs.NewAdmin(client, internalApi, log, namespace, constLabels),

		enabledJobs: make(map[string]bool),
	}

	if able := jobs.ExporterCanRun(enabledModules, m.syncMetrics.RequiredModules()); able {
		m.log.Info("Enabling sync status metrics")
		m.enabledJobs[m.syncMetrics.Name()] = true

		prometheus.MustRegister(m.syncMetrics.Percentage)
		prometheus.MustRegister(m.syncMetrics.StartingBlock)
		prometheus.MustRegister(m.syncMetrics.CurrentBlock)
		prometheus.MustRegister(m.syncMetrics.IsSyncing)
		prometheus.MustRegister(m.syncMetrics.HighestBlock)
	}

	if able := jobs.ExporterCanRun(enabledModules, m.generalMetrics.RequiredModules()); able {
		m.log.Info("Enabling general metrics")
		m.enabledJobs[m.generalMetrics.Name()] = true

		prometheus.MustRegister(m.generalMetrics.NetworkID)
		prometheus.MustRegister(m.generalMetrics.GasPrice)
		prometheus.MustRegister(m.generalMetrics.MostRecentBlockNumber)
		prometheus.MustRegister(m.generalMetrics.ChainID)
	}

	if able := jobs.ExporterCanRun(enabledModules, m.txpoolMetrics.RequiredModules()); able {
		m.log.Info("Enabling txpool metrics")
		m.enabledJobs[m.txpoolMetrics.Name()] = true

		prometheus.MustRegister(m.txpoolMetrics.Transactions)
	}

	if able := jobs.ExporterCanRun(enabledModules, m.adminMetrics.RequiredModules()); able {
		m.log.Info("Enabling admin metrics")
		m.enabledJobs[m.adminMetrics.Name()] = true

		prometheus.MustRegister(m.adminMetrics.TotalDifficulty)
		prometheus.MustRegister(m.adminMetrics.NodeInfo)
		prometheus.MustRegister(m.adminMetrics.Port)
		prometheus.MustRegister(m.adminMetrics.Peers)
	}

	return m
}

func (m *metrics) StartAsync(ctx context.Context) {
	if m.enabledJobs[m.syncMetrics.Name()] {
		go m.syncMetrics.Start(ctx)
	}

	if m.enabledJobs[m.generalMetrics.Name()] {
		go m.generalMetrics.Start(ctx)
	}

	if m.enabledJobs[m.txpoolMetrics.Name()] {
		go m.txpoolMetrics.Start(ctx)
	}

	if m.enabledJobs[m.adminMetrics.Name()] {
		go m.adminMetrics.Start(ctx)
	}

	m.log.Info("Started metrics exporter jobs")
}
