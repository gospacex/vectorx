# Graph Report - .  (2026-06-13)

## Corpus Check
- 41 files · ~17,055 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 169 nodes · 151 edges · 38 communities detected
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]

## God Nodes (most connected - your core abstractions)
1. `Milvusx` - 9 edges
2. `Weaviatex` - 6 edges
3. `Qdrantx` - 5 edges
4. `loadConfig()` - 4 edges
5. `getGlobalConfig()` - 4 edges
6. `buildTestSpans()` - 4 edges
7. `SetConfigPath()` - 3 edges
8. `newClient()` - 3 edges
9. `writeTestConfig()` - 3 edges
10. `streamExporter` - 3 edges

## Surprising Connections (you probably didn't know these)
- None detected - all connections are within the same source files.

## Communities

### Community 0 - "Community 0"
Cohesion: 0.13
Nodes (3): newClient(), Qdrantx, Weaviatex

### Community 1 - "Community 1"
Cohesion: 0.18
Nodes (5): fakePublisher, buildTestSpans(), TestStreamExporter_ExportSpans(), TestStreamExporter_KafkaUsesTopic(), TestStreamExporter_PublisherError()

### Community 2 - "Community 2"
Cohesion: 0.22
Nodes (1): publisher

### Community 3 - "Community 3"
Cohesion: 0.22
Nodes (1): publisher

### Community 4 - "Community 4"
Cohesion: 0.22
Nodes (1): testPublisher

### Community 5 - "Community 5"
Cohesion: 0.22
Nodes (1): Milvusx

### Community 6 - "Community 6"
Cohesion: 0.22
Nodes (3): SpanPublisher, spanRecord, streamExporter

### Community 7 - "Community 7"
Cohesion: 0.25
Nodes (0): 

### Community 8 - "Community 8"
Cohesion: 0.67
Nodes (3): getGlobalConfig(), loadConfig(), SetConfigPath()

### Community 9 - "Community 9"
Cohesion: 0.47
Nodes (3): TestGetQdrant_UnknownName_ReturnsError(), TestMustGetQdrant_UnknownName_Panics(), writeTestConfig()

### Community 10 - "Community 10"
Cohesion: 0.47
Nodes (3): TestGetMilvus_UnknownName_ReturnsError(), TestMustGetMilvus_UnknownName_Panics(), writeTestConfig()

### Community 11 - "Community 11"
Cohesion: 0.47
Nodes (3): TestGetWeaviate_UnknownName_ReturnsError(), TestMustGetWeaviate_UnknownName_Panics(), writeTestConfig()

### Community 12 - "Community 12"
Cohesion: 0.4
Nodes (3): Config, MQXSection, VectorXSection

### Community 13 - "Community 13"
Cohesion: 0.4
Nodes (0): 

### Community 14 - "Community 14"
Cohesion: 0.67
Nodes (2): InitTracing(), samplerFromConfig()

### Community 15 - "Community 15"
Cohesion: 0.5
Nodes (1): publisherError

### Community 16 - "Community 16"
Cohesion: 1.0
Nodes (2): fieldNamesOf(), TestTracingConfig_FieldNamesMatchMQX()

### Community 17 - "Community 17"
Cohesion: 0.67
Nodes (1): TracingConfig

### Community 18 - "Community 18"
Cohesion: 1.0
Nodes (2): GetQdrant(), MustGetQdrant()

### Community 19 - "Community 19"
Cohesion: 0.67
Nodes (0): 

### Community 20 - "Community 20"
Cohesion: 1.0
Nodes (2): GetMilvus(), MustGetMilvus()

### Community 21 - "Community 21"
Cohesion: 1.0
Nodes (2): GetWeaviate(), MustGetWeaviate()

### Community 22 - "Community 22"
Cohesion: 1.0
Nodes (1): QdrantConfig

### Community 23 - "Community 23"
Cohesion: 1.0
Nodes (1): MilvusConfig

### Community 24 - "Community 24"
Cohesion: 1.0
Nodes (0): 

### Community 25 - "Community 25"
Cohesion: 1.0
Nodes (1): WeaviateConfig

### Community 26 - "Community 26"
Cohesion: 1.0
Nodes (0): 

### Community 27 - "Community 27"
Cohesion: 1.0
Nodes (0): 

### Community 28 - "Community 28"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "Community 29"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "Community 30"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (0): 

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (0): 

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **8 isolated node(s):** `QdrantConfig`, `MilvusConfig`, `WeaviateConfig`, `Config`, `MQXSection` (+3 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 22`** (2 nodes): `qdrant.go`, `QdrantConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 23`** (2 nodes): `milvus.go`, `MilvusConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 24`** (2 nodes): `load_test.go`, `TestLoad_ParseVectorXBlock()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 25`** (2 nodes): `weaviate.go`, `WeaviateConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 26`** (2 nodes): `TestWeaviateLiveE2E()`, `e2e_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 27`** (2 nodes): `TestQdrantSearchE2E()`, `e2e_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 28`** (2 nodes): `TestMilvusSearchE2E()`, `e2e_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 29`** (2 nodes): `tracing_test.go`, `TestStartSpan_Disabled_NoOp()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 30`** (2 nodes): `TestObservability_Deps_MustNotContainAdapterPackages()`, `deps_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (2 nodes): `buildKafkaExporter()`, `kafka.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `buildOTLP()`, `jaeger.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (1 nodes): `tracing.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (1 nodes): `metrics.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (1 nodes): `tracing.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (1 nodes): `tracing.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Milvusx` connect `Community 5` to `Community 0`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **What connects `QdrantConfig`, `MilvusConfig`, `WeaviateConfig` to the rest of the system?**
  _8 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.13 - nodes in this community are weakly interconnected._