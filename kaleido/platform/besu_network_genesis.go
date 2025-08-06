package platform

import (
	"context"
	"encoding/json"
	"math/big"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rlp"
)

func BesuQBFTNetworkGenesisDatasourceFactory() datasource.DataSource {
	return &besuQBFTNetworkGenesisDatasource{}
}

type besuQBFTNetworkGenesisDatasource struct {
	commonDataSource
}

func (r *besuQBFTNetworkGenesisDatasource) Metadata(ctx context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "kaleido_platform_besu_qbft_network_genesis"
}

func (r *besuQBFTNetworkGenesisDatasource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Besu network genesis is a genesis file for a Besu network.",
		Attributes: map[string]schema.Attribute{
			"chain_id": &schema.Int64Attribute{
				Description: "The chain ID for the Besu network. This is used to identify the network and is used to prevent replay attacks.",
				Required:    true,
			},
			"validator_keys": &schema.ListAttribute{
				Description: "The validator signing key addresses for the Besu network. These are the addresses that will be encoded into the QBFT extra data and used to sign the initial blocks.",
				ElementType: types.StringType,
				Required:    true,
			},
			"qbft_config": &schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"block_period": &schema.Int64Attribute{
						Description: "The block period in seconds for the Besu network. This is the time between blocks.",
						Optional:    true,
						Computed:    true,
					},
					"epoch": &schema.Int64Attribute{
						Description: "The epoch length for the Besu network. This is the number of blocks in an epoch.",
						Optional:    true,
						Computed:    true,
					},
					"request_timeout": &schema.Int64Attribute{
						Description: "The request timeout in seconds for the Besu network. This is the time after which a request will be timed out.",
						Optional:    true,
						Computed:    true,
					},
				},
				Optional: true,
				Computed: true,
			},

			// TODO alloc, gas, etc.

			"genesis": &schema.StringAttribute{
				Description: "The genesis JSON file for the Besu network.",
				Computed:    true,
			},
		},
	}
}

type besuGenesis struct {
	Alloc      alloc                     `json:"alloc"`
	CoinBase   ethtypes.Address0xHex     `json:"coinbase"`
	Config     genesisConfig             `json:"config"`
	Difficulty ethtypes.HexInteger       `json:"difficulty,omitempty"`
	ExtraData  ethtypes.HexBytes0xPrefix `json:"extraData,omitempty"`
	GasLimit   *ethtypes.HexInteger      `json:"gasLimit,omitempty"`
	MixHash    ethtypes.HexBytes0xPrefix `json:"mixHash,omitempty"` //in the operator version this was spelled mixhash (which also works) but mixHash is how it is documented
	Nonce      *ethtypes.HexInteger      `json:"nonce,omitempty"`
	ParentHash ethtypes.HexBytes0xPrefix `json:"parentHash,omitempty"`
	Timestamp  *ethtypes.HexInteger      `json:"timestamp,omitempty"`
}

type alloc map[string]*allocEntry

type allocEntry struct {
	Balance string `json:"balance"`
}

// GenesisConfig section preserves arbitrary data on serialization
type genesisConfig struct {
	ChainID           int64              `json:"chainId"`
	BerlinBlock       int64              `json:"berlinBlock"`
	ContractSizeLimit int64              `json:"contractSizeLimit"`
	ShanghaiTime      int64              `json:"shanghaiTime"`
	ZeroBaseFee       bool               `json:"zeroBaseFee"`
	QBFT              *qbftGenesisConfig `json:"qbft,omitempty"`
}

type qbftGenesisConfig struct {
	BlockPeriodSeconds    int64  `json:"blockperiodseconds,omitempty"`
	EpochLength           int64  `json:"epochlength,omitempty"`
	RequestTimeoutSeconds int64  `json:"requesttimeoutseconds,omitempty"`
	BlockReward           string `json:"blockreward,omitempty"`
	MiningBeneficiary     string `json:"miningbeneficiary,omitempty"`
}

type besuQBFTNetworkGenesisDatasourceModel struct {
	ChainID       types.Int64              `tfsdk:"chain_id"`
	ValidatorKeys types.List               `tfsdk:"validator_keys"`
	QBFTConfig    *besuQBFTDatasourceModel `tfsdk:"qbft_config"`

	Genesis types.String `tfsdk:"genesis"`
}

type besuQBFTDatasourceModel struct {
	BlockPeriod    types.Int64 `tfsdk:"block_period"`
	Epoch          types.Int64 `tfsdk:"epoch"`
	RequestTimeout types.Int64 `tfsdk:"request_timeout"`
}

const QBFT_MIX_HASH = "0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365"
const DEFAULT_TARGET_GAS_LIMIT = int64(804247552)
const DEFAULT_BLOCK_PERIOD_SECONDS = int64(5)
const DEFAULT_EPOCH = int64(30000)
const DEFAULT_REQUEST_TIMEOUT_SECONDS = int64(10)
const DEFAULT_CONTRACT_SIZE_LIMIT = int64(98304)

func (r *besuQBFTNetworkGenesisDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data besuQBFTNetworkGenesisDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validatorKeys := data.ValidatorKeys.Elements()
	validatorAddresses := make([]string, len(validatorKeys))
	for i, key := range validatorKeys {
		var err error
		keyValue, err := key.ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError("failed to convert validator key to terraform value", err.Error())
			return
		}
		err = keyValue.As(&validatorAddresses[i])
		if err != nil {
			resp.Diagnostics.AddError("failed to convert validator key to string", err.Error())
			return
		}
	}

	slices.Sort(validatorAddresses)

	extraData := createQBFTBesuExtraData(validatorAddresses...)

	genesis := besuGenesis{
		GasLimit:   ethtypes.NewHexInteger64(DEFAULT_TARGET_GAS_LIMIT),
		Difficulty: *ethtypes.NewHexInteger64(1),
		Alloc:      make(alloc),
		MixHash:    ethtypes.MustNewHexBytes0xPrefix(QBFT_MIX_HASH),
		ExtraData:  extraData,
		Config: genesisConfig{
			ChainID:           data.ChainID.ValueInt64(),
			BerlinBlock:       int64(0),
			ContractSizeLimit: int64(DEFAULT_CONTRACT_SIZE_LIMIT),
			// TODO make EIP blocks and gas available as config options
			ShanghaiTime: int64(0),
			ZeroBaseFee:  true,
			QBFT:         &qbftGenesisConfig{},
		},
	}

	qbftConfig := qbftGenesisConfig{}
	if data.QBFTConfig == nil {
		data.QBFTConfig = &besuQBFTDatasourceModel{}
	}

	if data.QBFTConfig.BlockPeriod.IsNull() {
		qbftConfig.BlockPeriodSeconds = DEFAULT_BLOCK_PERIOD_SECONDS
	} else {
		qbftConfig.BlockPeriodSeconds = data.QBFTConfig.BlockPeriod.ValueInt64()
	}

	if data.QBFTConfig.Epoch.IsNull() {
		qbftConfig.EpochLength = DEFAULT_EPOCH
	} else {
		qbftConfig.EpochLength = data.QBFTConfig.Epoch.ValueInt64()
	}

	if data.QBFTConfig.RequestTimeout.IsNull() {
		qbftConfig.RequestTimeoutSeconds = DEFAULT_REQUEST_TIMEOUT_SECONDS
	} else {
		qbftConfig.RequestTimeoutSeconds = data.QBFTConfig.RequestTimeout.ValueInt64()
	}
	genesis.Config.QBFT = &qbftConfig

	genesisBytes, err := json.Marshal(genesis)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal genesis", err.Error())
		return
	}

	resp.State.SetAttribute(ctx, path.Root("genesis"), string(genesisBytes))
}

func createQBFTBesuExtraData(keys ...string) ethtypes.HexBytes0xPrefix {
	// Besu extraData is made up of:

	// 32-bytes of vanity data
	vanity := rlp.Data(make([]byte, 32))

	// List of genesis validators
	validatorList := rlp.List{}
	for _, key := range keys {
		validatorList = append(validatorList, rlp.WrapAddress(ethtypes.MustNewAddress(key)))
	}

	// Zero-length list of votes
	emptyVoteList := rlp.List{}

	// Vote round (0)
	roundZero := rlp.WrapInt(big.NewInt(0))

	// Zero-length list of seals
	emptySealsList := rlp.List{}

	rlpList := rlp.List{}
	rlpList = append(rlpList, vanity)
	rlpList = append(rlpList, validatorList)
	rlpList = append(rlpList, emptyVoteList)
	rlpList = append(rlpList, roundZero)
	rlpList = append(rlpList, emptySealsList)

	return ethtypes.HexBytes0xPrefix(rlpList.Encode())
}
