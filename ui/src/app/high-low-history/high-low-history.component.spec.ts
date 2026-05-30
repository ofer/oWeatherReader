import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { MatTableModule } from '@angular/material/table';
import { CommonModule } from '@angular/common';

import { HighLowHistoryComponent } from './high-low-history.component';

describe('HighLowHistoryComponent', () => {
  let component: HighLowHistoryComponent;
  let fixture: ComponentFixture<HighLowHistoryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [HighLowHistoryComponent],
      imports: [
        HttpClientTestingModule,
        MatTableModule,
        CommonModule,
      ]
    });
    fixture = TestBed.createComponent(HighLowHistoryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
