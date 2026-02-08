package domain

import (
	"fmt"

	paymentuc "github.com/erniealice/espyna-golang/internal/application/usecases/payment"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
	// paymentprofilepaymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile_payment_method" // TODO: Uncomment when PaymentProfilePaymentMethod is implemented
)

// ConfigurePaymentDomain configures routes for the Payment domain with use cases injected directly
func ConfigurePaymentDomain(paymentUseCases *paymentuc.PaymentUseCases) contracts.DomainRouteConfiguration {
	// Handle nil use cases gracefully for backward compatibility
	if paymentUseCases == nil {
		fmt.Printf("⚠️  Payment use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "payment",
			Prefix:  "/payment",
			Enabled: false,                            // Disable until use cases are properly initialized
			Routes:  []contracts.RouteConfiguration{}, // No routes without use cases
		}
	}

	fmt.Printf("✅ Payment use cases are properly initialized!\n")

	// Initialize routes array
	routes := []contracts.RouteConfiguration{}

	// Payment module routes
	if paymentUseCases.Payment != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment/create",
			Handler: contracts.NewGenericHandler(paymentUseCases.Payment.CreatePayment, &paymentpb.CreatePaymentRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment/read",
			Handler: contracts.NewGenericHandler(paymentUseCases.Payment.ReadPayment, &paymentpb.ReadPaymentRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment/update",
			Handler: contracts.NewGenericHandler(paymentUseCases.Payment.UpdatePayment, &paymentpb.UpdatePaymentRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment/delete",
			Handler: contracts.NewGenericHandler(paymentUseCases.Payment.DeletePayment, &paymentpb.DeletePaymentRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment/list",
			Handler: contracts.NewGenericHandler(paymentUseCases.Payment.ListPayments, &paymentpb.ListPaymentsRequest{}),
		})

		// TODO: Uncomment when GetPaymentListPageData and GetPaymentItemPageData are implemented
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.Payment.GetPaymentListPageData, &paymentpb.GetPaymentListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.Payment.GetPaymentItemPageData, &paymentpb.GetPaymentItemPageDataRequest{}),
		// })
	}

	// PaymentAttribute module routes
	if paymentUseCases.PaymentAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/create",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.CreatePaymentAttribute, &paymentattributepb.CreatePaymentAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/read",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.ReadPaymentAttribute, &paymentattributepb.ReadPaymentAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/update",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.UpdatePaymentAttribute, &paymentattributepb.UpdatePaymentAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/delete",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.DeletePaymentAttribute, &paymentattributepb.DeletePaymentAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/list",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.ListPaymentAttributes, &paymentattributepb.ListPaymentAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.GetPaymentAttributeListPageData, &paymentattributepb.GetPaymentAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentAttribute.GetPaymentAttributeItemPageData, &paymentattributepb.GetPaymentAttributeItemPageDataRequest{}),
		})
	}

	// PaymentMethod module routes
	if paymentUseCases.PaymentMethod != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-method/create",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.CreatePaymentMethod, &paymentmethodpb.CreatePaymentMethodRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-method/read",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.ReadPaymentMethod, &paymentmethodpb.ReadPaymentMethodRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-method/update",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.UpdatePaymentMethod, &paymentmethodpb.UpdatePaymentMethodRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-method/delete",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.DeletePaymentMethod, &paymentmethodpb.DeletePaymentMethodRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-method/list",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.ListPaymentMethods, &paymentmethodpb.ListPaymentMethodsRequest{}),
		})

		// TODO: Uncomment when GetPaymentMethodListPageData and GetPaymentMethodItemPageData are implemented
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment-method/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.GetPaymentMethodListPageData, &paymentmethodpb.GetPaymentMethodListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment-method/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.PaymentMethod.GetPaymentMethodItemPageData, &paymentmethodpb.GetPaymentMethodItemPageDataRequest{}),
		// })
	}

	// PaymentProfile module routes
	if paymentUseCases.PaymentProfile != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-profile/create",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.CreatePaymentProfile, &paymentprofilepb.CreatePaymentProfileRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-profile/read",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.ReadPaymentProfile, &paymentprofilepb.ReadPaymentProfileRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-profile/update",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.UpdatePaymentProfile, &paymentprofilepb.UpdatePaymentProfileRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-profile/delete",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.DeletePaymentProfile, &paymentprofilepb.DeletePaymentProfileRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/payment/payment-profile/list",
			Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.ListPaymentProfiles, &paymentprofilepb.ListPaymentProfilesRequest{}),
		})

		// TODO: Uncomment when GetPaymentProfileListPageData and GetPaymentProfileItemPageData are implemented
		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment-profile/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.GetPaymentProfileListPageData, &paymentprofilepb.GetPaymentProfileListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/payment/payment-profile/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(paymentUseCases.PaymentProfile.GetPaymentProfileItemPageData, &paymentprofilepb.GetPaymentProfileItemPageDataRequest{}),
		// })
	}

	// TODO: Add PaymentProfilePaymentMethod module when implemented
	// if paymentUseCases.PaymentProfilePaymentMethod != nil {
	// 	// Add PaymentProfilePaymentMethod routes here
	// }

	return contracts.DomainRouteConfiguration{
		Domain:  "payment",
		Prefix:  "/payment",
		Enabled: true,
		Routes:  routes,
	}
}
