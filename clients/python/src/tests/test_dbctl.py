from dbctl import dbctl
import unittest

class TestDBCtl(unittest.TestCase):
    def test_dbctl(self):
        uri = dbctl.must_create_database(
            database_type=dbctl.DATABASE_POSTGRES,
            config=dbctl.Config(
                migrations="../test_sql/schema",
                fixtures="../test_sql/fixtures",
            ))
        
        self.assertIsNotNone(uri)
        print(uri)

        dbctl.remove_database(
            database_type=dbctl.DATABASE_POSTGRES,
            uri=uri)
        
    
if __name__ == '__main__':
    unittest.main()
