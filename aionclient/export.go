// Package aionclient is a deprecated compatibility shim for the I/O Mesh Go client.
//
// Prefer:
//
//	import "github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
//
// This package re-exports iomeshclient for one minor series so existing
// imports keep compiling. New code must use iomeshclient. Wire headers
// (X-Aion-Tenant / X-Aion-Org) are unchanged broker protocol names.
//
// Deprecated: use package iomeshclient.
package aionclient

import (
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

const (
	ProcessorTypeEnrich = iomeshclient.ProcessorTypeEnrich
	ProcessorTypeFilter = iomeshclient.ProcessorTypeFilter
	ProcessorTypeMap    = iomeshclient.ProcessorTypeMap
	RetentionLimits     = iomeshclient.RetentionLimits
	RetentionWorkQueue  = iomeshclient.RetentionWorkQueue
)

type (
	APIError            = iomeshclient.APIError
	Client              = iomeshclient.Client
	ConnectOpt          = iomeshclient.ConnectOpt
	CreateBucketConfig  = iomeshclient.CreateBucketConfig
	DataProduct         = iomeshclient.DataProduct
	FetchOpt            = iomeshclient.FetchOpt
	IcebergCatalogRef   = iomeshclient.IcebergCatalogRef
	KafkaClient         = iomeshclient.KafkaClient
	KVEntry             = iomeshclient.KVEntry
	LiveView            = iomeshclient.LiveView
	MemoryEnvelope      = iomeshclient.MemoryEnvelope
	MemoryProductConfig = iomeshclient.MemoryProductConfig
	MeshSDK             = iomeshclient.MeshSDK
	MeshSDKConfig       = iomeshclient.MeshSDKConfig
	Msg                 = iomeshclient.Msg
	Options             = iomeshclient.Options
	ProcessorConfig     = iomeshclient.ProcessorConfig
	PubAck              = iomeshclient.PubAck
	PublishOpt          = iomeshclient.PublishOpt
	PullSubscribeConfig = iomeshclient.PullSubscribeConfig
	StreamConfig        = iomeshclient.StreamConfig
	SubsetConfig        = iomeshclient.SubsetConfig
	Subscription        = iomeshclient.Subscription
)

// Connect is a deprecated alias for iomeshclient.Connect.
func Connect(base Options, opts ...ConnectOpt) (*Client, error) {
	return iomeshclient.Connect(base, opts...)
}

// MaxWait is a deprecated alias for iomeshclient.MaxWait.
func MaxWait(d time.Duration) FetchOpt { return iomeshclient.MaxWait(d) }

// NewKafkaClient is a deprecated alias for iomeshclient.NewKafkaClient.
func NewKafkaClient(addr string) *KafkaClient { return iomeshclient.NewKafkaClient(addr) }

// NewMeshSDK is a deprecated alias for iomeshclient.NewMeshSDK.
func NewMeshSDK(cfg MeshSDKConfig, opts ...ConnectOpt) (*MeshSDK, error) {
	return iomeshclient.NewMeshSDK(cfg, opts...)
}

// WithBearerToken is a deprecated alias for iomeshclient.WithBearerToken.
func WithBearerToken(token string) ConnectOpt { return iomeshclient.WithBearerToken(token) }

// WithOrg is a deprecated alias for iomeshclient.WithOrg.
func WithOrg(orgID string) ConnectOpt { return iomeshclient.WithOrg(orgID) }

// WithPartition is a deprecated alias for iomeshclient.WithPartition.
func WithPartition(p int) PublishOpt { return iomeshclient.WithPartition(p) }

// WithPartitionKey is a deprecated alias for iomeshclient.WithPartitionKey.
func WithPartitionKey(key string) PublishOpt { return iomeshclient.WithPartitionKey(key) }

// WithTenant is a deprecated alias for iomeshclient.WithTenant.
func WithTenant(tenant string) ConnectOpt { return iomeshclient.WithTenant(tenant) }
