#!/bin/bash
# Setup MariaDB for OMA OSSEA integration

set -e

echo "🔧 Setting up MariaDB for OMA OSSEA integration with VMA enrollment..."

# Database configuration
DB_NAME="migratekit_oma"
DB_USER="oma_user"
DB_PASSWORD="oma_password"  # Using the standard password from the project
CHARSET="utf8mb4"

# Create SQL commands file
cat << EOF > /tmp/oma_mariadb_setup.sql
-- Create database
CREATE DATABASE IF NOT EXISTS ${DB_NAME} CHARACTER SET ${CHARSET} COLLATE ${CHARSET}_general_ci;

-- Create user (drop if exists first to avoid errors)
DROP USER IF EXISTS '${DB_USER}'@'localhost';
CREATE USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASSWORD}';

-- Grant privileges
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';

-- Also create user for % host for future remote access if needed
DROP USER IF EXISTS '${DB_USER}'@'%';
CREATE USER '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASSWORD}';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';

-- Flush privileges
FLUSH PRIVILEGES;

-- Show results
SELECT User, Host FROM mysql.user WHERE User = '${DB_USER}';
SHOW DATABASES LIKE '${DB_NAME}';
EOF

echo "📝 SQL script created at /tmp/oma_mariadb_setup.sql"

# Execute the SQL commands
echo "🔄 Creating database and user..."
sudo mysql < /tmp/oma_mariadb_setup.sql

# Test the connection
echo "🧪 Testing database connection..."
mysql -u${DB_USER} -p${DB_PASSWORD} -e "SELECT DATABASE();" ${DB_NAME}

# Create tables from our schema
echo "🔄 Creating database schema..."
if [ -f /home/pgrayson/migratekit-cloudstack/internal/oma/database/migrations/20250115120000_initial_schema.up.sql ]; then
    mysql -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} < /home/pgrayson/migratekit-cloudstack/internal/oma/database/migrations/20250115120000_initial_schema.up.sql
    echo "✅ Database schema created successfully"
else
    echo "⚠️  Migration file not found. You'll need to run migrations manually."
fi

# Show created tables
echo "📊 Created tables:"
mysql -u${DB_USER} -p${DB_PASSWORD} -e "SHOW TABLES;" ${DB_NAME}

# Create environment file for OMA
cat << EOF > /home/pgrayson/oma_mariadb.env
# MariaDB configuration for OMA - Production Database
export MARIADB_HOST=localhost
export MARIADB_PORT=3306
export MARIADB_DATABASE=${DB_NAME}
export MARIADB_USERNAME=${DB_USER}
export MARIADB_PASSWORD=${DB_PASSWORD}
export MARIADB_CHARSET=${CHARSET}
EOF

echo "✅ MariaDB setup complete!"
echo ""
echo "📋 Connection details:"
echo "   Host: localhost"
echo "   Port: 3306"
echo "   Database: ${DB_NAME}"
echo "   Username: ${DB_USER}"
echo "   Password: ${DB_PASSWORD}"
echo ""
echo "🔐 Environment file created at: /home/pgrayson/oma_mariadb.env"
echo "   Source it with: source /home/pgrayson/oma_mariadb.env"
echo ""
echo "🧪 Test connection with:"
echo "   mysql -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME}"

# Clean up
rm -f /tmp/oma_mariadb_setup.sql