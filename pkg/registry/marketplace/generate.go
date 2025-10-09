//go:generate go run go.uber.org/mock/mockgen -source=../../../vendor/github.com/digitalocean/godo/marketplace.go -destination=mocks.go -package=marketplace -mock_names=OneClickService=MockOneClickService

package marketplace
