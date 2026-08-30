# Funclen Violations — TypeScript Example

## Problem

A function or method that's too long mixes multiple concerns and is hard to read and test.

```typescript
function processUserPayment(userId: string, amount: number): Promise<void> {
    // Fetch user
    const user = await database.getUser(userId);
    if (!user) throw new Error("User not found");

    // Validate payment
    if (amount <= 0) throw new Error("Invalid amount");
    if (amount > user.creditLimit) throw new Error("Insufficient credit");

    // Create payment record
    const payment = {
        userId,
        amount,
        timestamp: new Date(),
        status: "pending"
    };
    await database.createPayment(payment);

    // Process with payment gateway
    const result = await paymentGateway.charge({
        amount,
        currency: "USD",
        customerId: user.externalId
    });

    if (result.success) {
        await database.updatePayment(payment.id, { status: "completed" });
        await email.sendConfirmation(user.email, amount);
    } else {
        await database.updatePayment(payment.id, { status: "failed" });
        throw new Error(`Payment failed: ${result.error}`);
    }
}
```

This function does fetching, validation, database updates, and email notifications — too many jobs.

## Solution

Extract each concern into its own function:

```typescript
async function processUserPayment(userId: string, amount: number): Promise<void> {
    const user = await validateUser(userId);
    validatePaymentAmount(amount, user.creditLimit);

    const payment = await createPaymentRecord(userId, amount);
    const result = await chargePaymentGateway(user, amount);

    if (result.success) {
        await completePayment(payment, user);
    } else {
        await failPayment(payment, result);
    }
}

async function validateUser(userId: string) {
    const user = await database.getUser(userId);
    if (!user) throw new Error("User not found");
    return user;
}

function validatePaymentAmount(amount: number, limit: number) {
    if (amount <= 0) throw new Error("Invalid amount");
    if (amount > limit) throw new Error("Insufficient credit");
}

async function createPaymentRecord(userId: string, amount: number) {
    return database.createPayment({ userId, amount, timestamp: new Date(), status: "pending" });
}

async function completePayment(payment: Payment, user: User) {
    await database.updatePayment(payment.id, { status: "completed" });
    await email.sendConfirmation(user.email, payment.amount);
}
```

Now each function has one clear job.
