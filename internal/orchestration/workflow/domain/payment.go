package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterPaymentUseCases registers all payment domain use cases with the registry.
func RegisterPaymentUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Payment == nil {
		return
	}

	registerPaymentCoreUseCases(useCases, register)
	registerPaymentMethodUseCases(useCases, register)
	registerPaymentProfileUseCases(useCases, register)
	registerPaymentAttributeUseCases(useCases, register)
}

func registerPaymentCoreUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Payment.Payment == nil {
		return
	}
	if useCases.Payment.Payment.CreatePayment != nil {
		register("payment.payment.create", executor.New(useCases.Payment.Payment.CreatePayment.Execute))
	}
	if useCases.Payment.Payment.ReadPayment != nil {
		register("payment.payment.read", executor.New(useCases.Payment.Payment.ReadPayment.Execute))
	}
	if useCases.Payment.Payment.UpdatePayment != nil {
		register("payment.payment.update", executor.New(useCases.Payment.Payment.UpdatePayment.Execute))
	}
	if useCases.Payment.Payment.DeletePayment != nil {
		register("payment.payment.delete", executor.New(useCases.Payment.Payment.DeletePayment.Execute))
	}
	if useCases.Payment.Payment.ListPayments != nil {
		register("payment.payment.list", executor.New(useCases.Payment.Payment.ListPayments.Execute))
	}
}

func registerPaymentMethodUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Payment.PaymentMethod == nil {
		return
	}
	if useCases.Payment.PaymentMethod.CreatePaymentMethod != nil {
		register("payment.payment_method.create", executor.New(useCases.Payment.PaymentMethod.CreatePaymentMethod.Execute))
	}
	if useCases.Payment.PaymentMethod.ReadPaymentMethod != nil {
		register("payment.payment_method.read", executor.New(useCases.Payment.PaymentMethod.ReadPaymentMethod.Execute))
	}
	if useCases.Payment.PaymentMethod.UpdatePaymentMethod != nil {
		register("payment.payment_method.update", executor.New(useCases.Payment.PaymentMethod.UpdatePaymentMethod.Execute))
	}
	if useCases.Payment.PaymentMethod.DeletePaymentMethod != nil {
		register("payment.payment_method.delete", executor.New(useCases.Payment.PaymentMethod.DeletePaymentMethod.Execute))
	}
	if useCases.Payment.PaymentMethod.ListPaymentMethods != nil {
		register("payment.payment_method.list", executor.New(useCases.Payment.PaymentMethod.ListPaymentMethods.Execute))
	}
}

func registerPaymentProfileUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Payment.PaymentProfile == nil {
		return
	}
	if useCases.Payment.PaymentProfile.CreatePaymentProfile != nil {
		register("payment.payment_profile.create", executor.New(useCases.Payment.PaymentProfile.CreatePaymentProfile.Execute))
	}
	if useCases.Payment.PaymentProfile.ReadPaymentProfile != nil {
		register("payment.payment_profile.read", executor.New(useCases.Payment.PaymentProfile.ReadPaymentProfile.Execute))
	}
	if useCases.Payment.PaymentProfile.UpdatePaymentProfile != nil {
		register("payment.payment_profile.update", executor.New(useCases.Payment.PaymentProfile.UpdatePaymentProfile.Execute))
	}
	if useCases.Payment.PaymentProfile.DeletePaymentProfile != nil {
		register("payment.payment_profile.delete", executor.New(useCases.Payment.PaymentProfile.DeletePaymentProfile.Execute))
	}
	if useCases.Payment.PaymentProfile.ListPaymentProfiles != nil {
		register("payment.payment_profile.list", executor.New(useCases.Payment.PaymentProfile.ListPaymentProfiles.Execute))
	}
}

func registerPaymentAttributeUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Payment.PaymentAttribute == nil {
		return
	}
	if useCases.Payment.PaymentAttribute.CreatePaymentAttribute != nil {
		register("payment.payment_attribute.create", executor.New(useCases.Payment.PaymentAttribute.CreatePaymentAttribute.Execute))
	}
	if useCases.Payment.PaymentAttribute.CreatePaymentAttributesByCode != nil {
		register("payment.payment_attribute.create_by_code", executor.New(useCases.Payment.PaymentAttribute.CreatePaymentAttributesByCode.Execute))
	}
	if useCases.Payment.PaymentAttribute.ReadPaymentAttribute != nil {
		register("payment.payment_attribute.read", executor.New(useCases.Payment.PaymentAttribute.ReadPaymentAttribute.Execute))
	}
	if useCases.Payment.PaymentAttribute.UpdatePaymentAttribute != nil {
		register("payment.payment_attribute.update", executor.New(useCases.Payment.PaymentAttribute.UpdatePaymentAttribute.Execute))
	}
	if useCases.Payment.PaymentAttribute.DeletePaymentAttribute != nil {
		register("payment.payment_attribute.delete", executor.New(useCases.Payment.PaymentAttribute.DeletePaymentAttribute.Execute))
	}
	if useCases.Payment.PaymentAttribute.ListPaymentAttributes != nil {
		register("payment.payment_attribute.list", executor.New(useCases.Payment.PaymentAttribute.ListPaymentAttributes.Execute))
	}
}
