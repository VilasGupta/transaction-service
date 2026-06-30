CREATE TABLE IF NOT EXISTS accounts (
    account_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    document_number VARCHAR(20) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS operation_types (
    operation_type_id INT PRIMARY KEY,
    description VARCHAR(50) NOT NULL
);

INSERT IGNORE INTO operation_types (operation_type_id, description) VALUES
    (1, 'Normal Purchase'),
    (2, 'Purchase with installments'),
    (3, 'Withdrawal'),
    (4, 'Credit Voucher');

CREATE TABLE IF NOT EXISTS transactions (
    transaction_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    account_id BIGINT NOT NULL,
    operation_type_id INT NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    balance DECIMAL(15,2) NOT NULL,
    event_date DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (account_id) REFERENCES accounts(account_id),
    FOREIGN KEY (operation_type_id) REFERENCES operation_types(operation_type_id)
);
