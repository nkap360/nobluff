"""The test suite that shipped with the handler. It passes, and the bug is still there."""
import unittest

import payments


class PaymentTest(unittest.TestCase):
    def test_successful_payment_is_booked(self):
        db = payments.connect()
        self.addCleanup(db.close)
        payments.handle_payment_succeeded(db, "evt_042", 14900)
        self.assertEqual(payments.balance_cents(db), 14900)


if __name__ == "__main__":
    unittest.main()
