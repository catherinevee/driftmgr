#!/usr/bin/env python3
"""
Fix DriftMgr Database Issue

This script fixes the database initialization issue that's causing
authentication manager failures in driftmgr.
"""

import os
import sqlite3
import subprocess
import sys
from pathlib import Path

def create_database_schema():
    """Create the SQLite database with proper schema"""
    db_path = "./driftmgr.db"
    
    print(f"🔧 Creating database at: {db_path}")
    
    try:
        # Connect to SQLite database (creates it if it doesn't exist)
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Create users table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS users (
                id TEXT PRIMARY KEY,
                username TEXT UNIQUE NOT NULL,
                password_hash TEXT NOT NULL,
                role TEXT NOT NULL,
                created_at DATETIME NOT NULL,
                last_login DATETIME,
                password_changed_at DATETIME NOT NULL,
                failed_login_attempts INTEGER DEFAULT 0,
                locked_until DATETIME,
                email TEXT,
                mfa_enabled BOOLEAN DEFAULT FALSE,
                mfa_secret TEXT
            )
        ''')
        
        # Create user_sessions table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS user_sessions (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                token_hash TEXT NOT NULL,
                created_at DATETIME NOT NULL,
                expires_at DATETIME NOT NULL,
                ip_address TEXT,
                user_agent TEXT,
                FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
            )
        ''')
        
        # Create audit_logs table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS audit_logs (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                user_id TEXT,
                action TEXT NOT NULL,
                resource TEXT,
                ip_address TEXT,
                user_agent TEXT,
                timestamp DATETIME NOT NULL,
                details TEXT,
                FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
            )
        ''')
        
        # Create password_policies table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS password_policies (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                min_length INTEGER DEFAULT 8,
                require_uppercase BOOLEAN DEFAULT TRUE,
                require_lowercase BOOLEAN DEFAULT TRUE,
                require_numbers BOOLEAN DEFAULT TRUE,
                require_special_chars BOOLEAN DEFAULT TRUE,
                max_age_days INTEGER DEFAULT 90,
                prevent_reuse_count INTEGER DEFAULT 5,
                lockout_threshold INTEGER DEFAULT 5,
                lockout_duration_minutes INTEGER DEFAULT 30
            )
        ''')
        
        # Insert default password policy
        cursor.execute('''
            INSERT OR IGNORE INTO password_policies 
            (id, min_length, require_uppercase, require_lowercase, require_numbers, 
             require_special_chars, max_age_days, prevent_reuse_count, 
             lockout_threshold, lockout_duration_minutes)
            VALUES (1, 8, TRUE, TRUE, TRUE, TRUE, 90, 5, 5, 30)
        ''')
        
        # Create a default admin user (for testing purposes)
        import datetime
        import hashlib
        
        admin_id = "admin-001"
        admin_username = "admin"
        admin_password = "admin123"  # In production, use a secure password
        password_hash = hashlib.sha256(admin_password.encode()).hexdigest()
        current_time = datetime.datetime.now().isoformat()
        
        cursor.execute('''
            INSERT OR IGNORE INTO users 
            (id, username, password_hash, role, created_at, password_changed_at)
            VALUES (?, ?, ?, ?, ?, ?)
        ''', (admin_id, admin_username, password_hash, "admin", current_time, current_time))
        
        conn.commit()
        conn.close()
        
        print("[OK] Database created successfully!")
        print(f"   - Database file: {db_path}")
        print(f"   - Default admin user: admin/admin123")
        
        return True
        
    except Exception as e:
        print(f"[ERROR] Failed to create database: {e}")
        return False

def create_config_directory():
    """Create config directory if it doesn't exist"""
    config_dir = Path("./configs")
    if not config_dir.exists():
        config_dir.mkdir()
        print(f"[OK] Created config directory: {config_dir}")

def test_driftmgr_connection():
    """Test if driftmgr can now connect properly"""
    print("\n🔍 Testing driftmgr connection...")
    
    try:
        result = subprocess.run(
            ['driftmgr', '--version'],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if result.returncode == 0:
            print("[OK] DriftMgr is now working properly!")
            print(f"   Output: {result.stdout.strip()}")
            return True
        else:
            print("[ERROR] DriftMgr still has issues:")
            print(f"   Error: {result.stderr.strip()}")
            return False
            
    except subprocess.TimeoutExpired:
        print("⏰ DriftMgr command timed out")
        return False
    except FileNotFoundError:
        print("[WARNING] DriftMgr executable not found in PATH")
        return False
    except Exception as e:
        print(f"💥 Unexpected error: {e}")
        return False

def test_credentials_command():
    """Test the credentials command specifically"""
    print("\n🔍 Testing credentials command...")
    
    try:
        result = subprocess.run(
            ['driftmgr', 'credentials', '--show'],
            capture_output=True,
            text=True,
            timeout=15
        )
        
        if result.returncode == 0:
            print("[OK] Credentials command works!")
            print(f"   Output: {result.stdout.strip()}")
            return True
        else:
            print("[ERROR] Credentials command still fails:")
            print(f"   Error: {result.stderr.strip()}")
            return False
            
    except Exception as e:
        print(f"💥 Error testing credentials: {e}")
        return False

def main():
    """Main function to fix the database issue"""
    print("🔧 DriftMgr Database Fix")
    print("=" * 50)
    print("This script fixes the database initialization issue")
    print("that's causing authentication manager failures.")
    print("=" * 50)
    
    # Step 1: Create config directory
    create_config_directory()
    
    # Step 2: Create database with proper schema
    if not create_database_schema():
        print("[ERROR] Failed to create database. Exiting.")
        sys.exit(1)
    
    # Step 3: Test driftmgr connection
    if test_driftmgr_connection():
        print("\n🎉 Database fix successful!")
        
        # Step 4: Test credentials command
        test_credentials_command()
        
        print("\n" + "=" * 50)
        print("[OK] FIX COMPLETED")
        print("=" * 50)
        print("The database has been created and driftmgr should now work.")
        print("You can now run the user simulation scripts successfully.")
        print("=" * 50)
    else:
        print("\n[ERROR] Database fix may not have resolved all issues.")
        print("Please check the error messages above.")

if __name__ == "__main__":
    main()
